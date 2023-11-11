package main

import (
	"bufio"
	"context"
	"flag"
	"reflect"
	"sync"

	"fmt"
	//"encoding/binary"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	pb "example.com/query"
	"example.com/util"
	"google.golang.org/grpc"
)

const (
	//address = "localhost:50051"
	leftAddress  = "localhost:50051"
	rightAddress = "localhost:50052"
	//address      = "localhost:50051"
	//eps          = 0.5
	//delta        = 1e-6
	FailureProbLog2 = 40
)

var DBSize uint64
var DBSeed uint64
var ChunkSize uint64
var SetSize uint64
var threadNum uint64
var serverAddr string
var LogFile *os.File
var str string
var ignoreOffline bool

func ReadConfigInfo() (uint64, uint64) {
	file, err := os.Open("config.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	line, _, err := reader.ReadLine()
	if err != nil {
		log.Fatal(err)
	}
	split := strings.Split(string(line), " ")
	var DBSize uint64
	var DBSeed uint64

	if DBSize, err = strconv.ParseUint(split[0], 10, 32); err != nil {
		log.Fatal(err)
	}
	if DBSeed, err = strconv.ParseUint(split[1], 10, 32); err != nil {
		log.Fatal(err)
	}

	log.Printf("%v %v", DBSize, DBSeed)

	return uint64(DBSize), uint64(DBSeed)
}

type LocalSet struct {
	tag             uint32 // the tag of the set
	parity          util.DBEntry
	programmedPoint uint64
	isProgrammed    bool
}

type LocalBackupSet struct {
	tag              uint32
	parityAfterPunct util.DBEntry
}

type LocalBackupSetGroup struct {
	consumed uint64
	sets     []LocalBackupSet
}

type LocalReplacementGroup struct {
	consumed uint64
	indices  []uint64
	value    []util.DBEntry
}

func primaryNumParam(Q float64, ChunkSize float64, target float64) uint64 {
	k := math.Ceil(math.Log(2)*(target) + math.Log(Q))
	log.Printf("k = %v", k)
	return uint64(k) * uint64(ChunkSize)
}

func FailProbBallIntoBins(ballNum uint64, binNum uint64, binSize uint64) float64 {
	log.Printf("ballNum = %v, binNum = %v, binSize = %v", ballNum, binNum, binSize)
	mean := float64(ballNum) / float64(binNum)
	c := float64(binSize)/mean - 1
	log.Printf("mean = %v, c = %v", mean, c)
	// chernoff exp(-(c^2)/(2+c) * mean)
	t := (mean * (c * c) / (2 + c)) * math.Log(2)
	t -= math.Log2(float64(binNum))
	log.Printf("t = %v", t)
	return t
}

func runPIRWithOneServer(leftClient pb.QueryServiceClient, DBSize uint64, DBSeed uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*100000000000))
	defer cancel()
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))
	masterKey := util.RandKey(rng)
	// expand the master key to a long key. Save some computation.
	longKey := util.GetLongKey((*util.PrfKey128)(&masterKey))
	totalQueryNum := uint64(math.Sqrt(float64(DBSize)) * math.Log(float64(DBSize)))
	plannedQueryNum := totalQueryNum
	log.Printf("totalQueryNum %v", totalQueryNum)

	// M1 in the paper
	localSetNum := primaryNumParam(float64(totalQueryNum), float64(ChunkSize), FailureProbLog2+1)
	// if localSetNum is not a mulitple of 4 then we need to add some padding
	localSetNum = (localSetNum + threadNum - 1) / threadNum * threadNum

	// M2 in the paper
	backupSetNumPerGroup := 3 * uint64(float64(totalQueryNum)/float64(SetSize))
	backupSetNumPerGroup = (backupSetNumPerGroup + threadNum - 1) / threadNum * threadNum

	totalBackupSetNum := backupSetNumPerGroup * SetSize

	if totalQueryNum > 1000 {
		totalQueryNum = 1000
	}

	// Setup Phase:
	// 		The client sends a simple query to the left server to fetch the whole DB
	//		When the client gets i-th chunk, it updates all local sets' parities, stores some replacements.
	//		It also needs to update the backup sets' parities
	//
	start := time.Now()

	// Initialize local sets and backup sets

	localSets := make([]LocalSet, localSetNum)
	localBackupSets := make([]LocalBackupSet, totalBackupSetNum)
	localCache := make(map[uint64]util.DBEntry)
	localMissElements := make(map[uint64]util.DBEntry)
	tagCounter := uint32(0)

	for j := uint64(0); j < localSetNum; j++ {
		localSets[j] = LocalSet{
			tag:             tagCounter,
			parity:          util.ZeroEntry(),
			isProgrammed:    false,
			programmedPoint: 0,
		}
		tagCounter += 1
	}

	LocalBackupSetGroups := make([]LocalBackupSetGroup, SetSize)
	LocalReplacementGroups := make([]LocalReplacementGroup, SetSize)

	for i := uint64(0); i < SetSize; i++ {
		LocalBackupSetGroups[i].consumed = 0
		LocalBackupSetGroups[i].sets = localBackupSets[i*backupSetNumPerGroup : (i+1)*backupSetNumPerGroup]

		LocalReplacementGroups[i].consumed = 0
		LocalReplacementGroups[i].indices = make([]uint64, backupSetNumPerGroup)
		LocalReplacementGroups[i].value = make([]util.DBEntry, backupSetNumPerGroup)
	}

	for j := uint64(0); j < SetSize; j++ {
		for k := uint64(0); k < backupSetNumPerGroup; k++ {
			LocalBackupSetGroups[j].sets[k] = LocalBackupSet{
				tag:              tagCounter,
				parityAfterPunct: util.ZeroEntry(),
			}
			tagCounter += 1
		}
	}

	// now fetch the whole DB
	log.Printf("Start fetching the whole DB")
	// print the size of LocalSet using reflection
	log.Printf("Every Local Set Size %v bytes", reflect.TypeOf(LocalSet{}).Size())
	log.Printf("Every Local Backup Set Size %v bytes", reflect.TypeOf(LocalBackupSet{}).Size())
	log.Printf("Local Set Num %v, Local Backup Set Num %v", localSetNum, totalBackupSetNum)
	log.Printf("Local Set Size %v bytes -----------------------", uint64(reflect.TypeOf(LocalSet{}).Size()))
	localStorageSize := float64(localSetNum*uint64(reflect.TypeOf(LocalSet{}).Size()) + (totalBackupSetNum * uint64(reflect.TypeOf(LocalBackupSet{}).Size())))
	localStorageSize = localStorageSize + float64(totalBackupSetNum)*(8+util.DBEntrySize) // the replacements
	log.Printf("Local Storage Size %v MB", localStorageSize/1024/1024)
	perQueryCommCost := float64(SetSize) * float64(8)               // upload cost
	perQueryCommCost = perQueryCommCost + float64(util.DBEntrySize) // download cost
	log.Printf("Per query communication cost %v kb", perQueryCommCost/1024)

	str = fmt.Sprintf("Every Local Set Size %v bytes\n", reflect.TypeOf(LocalSet{}).Size())
	LogFile.WriteString(str)

	str = fmt.Sprintf("Every Local Backup Set Size %v bytes\n", reflect.TypeOf(LocalBackupSet{}).Size())
	LogFile.WriteString(str)

	str = fmt.Sprintf("Local Set Num %v, Local Backup Set Num %v\n", localSetNum, totalBackupSetNum)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Local Storage Size %v MB\n", localStorageSize/1024/1024)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Per query communication cost %v kb\n", perQueryCommCost/1024)
	LogFile.WriteString(str)

	/***************This is the offline phase*******************/

	if !ignoreOffline {
		fechFullDBMsg := &pb.FetchFullDBMsg{Dummy: 1}
		stream, err := leftClient.FetchFullDB(ctx, fechFullDBMsg)

		if err != nil {
			log.Fatalf("failed to fetch full DB %v", err)
		}

		for i := uint64(0); i < SetSize; i++ {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("failed to receive chunk %v", err)
			}
			if i%1000 == 0 {
				log.Printf("received chunk %v", i)
			}
			hitMap := make([]bool, ChunkSize)

			// use multiple threads to parallelize the computation for the chunk

			var wg sync.WaitGroup
			wg.Add(int(threadNum))

			perTheadSetNum := (localSetNum+threadNum-1)/threadNum + 1 // make sure all sets are covered
			perThreadBackupNum := (totalBackupSetNum+threadNum-1)/threadNum + 1

			for tid := uint64(0); tid < threadNum; tid++ {
				startIndex := uint64(tid) * uint64(perTheadSetNum)
				endIndex := startIndex + uint64(perTheadSetNum)
				if endIndex > localSetNum {
					endIndex = localSetNum
				}

				startIndexBackup := uint64(tid) * uint64(perThreadBackupNum)
				endIndexBackup := startIndexBackup + uint64(perThreadBackupNum)
				if endIndexBackup > totalBackupSetNum {
					endIndexBackup = totalBackupSetNum
				}

				go func(start, end, start1, end1 uint64) {
					defer wg.Done()
					// Here, we update the parities for the primary hints
					for j := uint64(start); j < uint64(end); j++ {
						tmp := util.PRFEvalWithLongKeyAndTag(longKey, localSets[j].tag, i)
						offset := tmp & (ChunkSize - 1)
						hitMap[offset] = true
						util.DBEntryXorFromRaw(&localSets[j].parity, chunk.Chunk[offset*util.DBEntryLength:(offset+1)*util.DBEntryLength])
					}

					// Here, we update the parities for the backup hints
					for j := uint64(start1); j < uint64(end1); j++ {
						tmp := util.PRFEvalWithLongKeyAndTag(longKey, localBackupSets[j].tag, i)
						offset := tmp & (ChunkSize - 1)
						util.DBEntryXorFromRaw(&localBackupSets[j].parityAfterPunct, chunk.Chunk[offset*util.DBEntryLength:(offset+1)*util.DBEntryLength])
					}
				}(startIndex, endIndex, startIndexBackup, endIndexBackup)
			}

			wg.Wait()

			// If any element is not hit, then it is a local miss. We will save it in the local cache.
			// Most of the time, the local cache will be empty.
			for j := uint64(0); j < ChunkSize; j++ {
				if hitMap[j] == false {
					entry := util.DBEntryFromSlice(chunk.Chunk[j*util.DBEntryLength : (j+1)*util.DBEntryLength])
					localMissElements[j+i*ChunkSize] = entry
				}
			}

			// For the i-th group of backups, leave the i-th chunk as blank
			// To do that, we just xor the i-th chunk's value again
			for k := uint64(0); k < backupSetNumPerGroup; k++ {
				tag := LocalBackupSetGroups[i].sets[k].tag
				tmp := util.PRFEvalWithTag(&masterKey, tag, i)
				offset := tmp & (ChunkSize - 1)
				util.DBEntryXorFromRaw(&LocalBackupSetGroups[i].sets[k].parityAfterPunct, chunk.Chunk[offset*util.DBEntryLength:(offset+1)*util.DBEntryLength])
			}

			// store the replacement
			for k := uint64(0); k < backupSetNumPerGroup; k++ {
				// generate a random offset between 0 and ChunkSize - 1
				offset := rng.Uint64() & (ChunkSize - 1)
				LocalReplacementGroups[i].indices[k] = offset + i*ChunkSize
				LocalReplacementGroups[i].value[k] = util.DBEntryFromSlice(chunk.Chunk[offset*util.DBEntryLength : (offset+1)*util.DBEntryLength])
			}
		}
	}

	/************Offline phase ends here****************/

	elapsed := time.Since(start)
	offlineElapsed := elapsed
	offlineCommCost := float64(DBSize) * float64(reflect.TypeOf(util.DBEntry{}).Size())

	log.Printf("Finish Setup Phase, store %v local sets, %v backup sets/replacement pairs", localSetNum, SetSize*backupSetNumPerGroup)
	log.Printf("Local Storage Size %v MB", localStorageSize/1024/1024)
	log.Printf("Setup Phase took %v ms, amortized time %v ms per query", elapsed.Milliseconds(), float64(elapsed.Milliseconds())/float64(plannedQueryNum))
	log.Printf("Setup Phase Comm Cost %v MB, amortized cost %v KB per query", float64(offlineCommCost)/1024/1024, float64(offlineCommCost)/1024/float64(plannedQueryNum))
	log.Printf("Num of local miss elements %v", len(localMissElements))
	str = fmt.Sprintf("Finish Setup Phase, store %v local sets, %v backup sets\n", localSetNum, SetSize*backupSetNumPerGroup)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Local Storage Size %v MB\n", localStorageSize/1024/1024)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Setup Phase took %v ms, amortized time %v ms per query\n", elapsed.Milliseconds(), float64(elapsed.Milliseconds())/float64(totalQueryNum))
	LogFile.WriteString(str)

	str = fmt.Sprintf("Setup Phase Comm Cost %v MB, amortized cost %v KB per query", float64(offlineCommCost)/1024/1024, float64(offlineCommCost)/1024/float64(plannedQueryNum))
	LogFile.WriteString(str)

	str = fmt.Sprintf("Num of local miss elements %v\n", len(localMissElements))
	LogFile.WriteString(str)

	totalNetworkLatency := uint64(0)
	totalServerComputeTime := uint64(0)
	totalFindHintTime := uint64(0)

	start = time.Now()

	/*************** This is the online phase *******************/

	for q := uint64(0); q < totalQueryNum; q++ {
		if q%10000 == 0 {
			log.Printf("Making %v-th query", q)
		}
		// just do random query for now
		x := rng.Uint64() % DBSize

		// make sure x is not in the local cache
		for true {
			if _, ok := localCache[x]; ok == false {
				break
			}
			x = rng.Uint64() % DBSize
		}

		// 		1. Query x: the client first finds a local set that contains x
		// 		2. The client expands the set, replace the chunk(x)-th element to a replacement
		// 		3. The client sends the edited set to the server and gets the parity
		//      4. The client recovers the answer

		hitSetId := uint64(999999999)

		findHintStart := time.Now()

		queryOffset := x % ChunkSize
		chunkId := x / ChunkSize

		/******* Looking for the hit set *******/
		for i := uint64(0); i < localSetNum; i++ {
			//tmpKey := localSets[i].key
			//set := util.PRSet{Key: tmpKey}
			if localSets[i].isProgrammed && chunkId == (localSets[i].programmedPoint/ChunkSize) {
				if x == localSets[i].programmedPoint {
					log.Fatalf("should not happen x = %v, programmedPoint = %v", x, localSets[i].programmedPoint)
					hitSetId = i
					break
				}
			} else {
				if util.MembTestWithLongKeyAndTag(longKey, localSets[i].tag, chunkId, queryOffset, ChunkSize) {
					hitSetId = i
					break
				}
			}
		}
		/******* Found some hit set *******/

		findHintElapsed := time.Since(findHintStart)
		totalFindHintTime += uint64(findHintElapsed.Nanoseconds())

		/*  the parallelization of online phase doesn't bring much benefit
		// use 4 threads and each thread find an interval of size ChunkSize
		perThreadSetNum := ChunkSize // because finding a hit set is with prob 1/ChunkSize

		// for each 4*ChunkSize interval, use 4 threads to find a hit set
		// if not, then go to the next interval

		for startSetId := uint64(0); startSetId < localSetNum; startSetId += perThreadSetNum * 4 {
			var wg sync.WaitGroup
			wg.Add(4)

			results := make(chan uint64, 4)

			for tid := uint64(0); tid < 4; tid++ {
				startIndex := startSetId + tid*perThreadSetNum
				endIndex := startIndex + perThreadSetNum
				if endIndex > localSetNum {
					endIndex = localSetNum
				}
				go func(start, end uint64) {
					defer wg.Done()
					hitSetId := uint64(999999999)
					for i := uint64(start); i < end; i++ {
						tmpKey := localSets[i].key
						set := util.PRSet{Key: tmpKey}
						if localSets[i].isProgrammed && (x/ChunkSize) == (localSets[i].programmedPoint/ChunkSize) {
							if x == localSets[i].programmedPoint {
								log.Fatalf("should not happen x = %v, programmedPoint = %v", x, localSets[i].programmedPoint)
								hitSetId = i
								break
							}
						} else {
							if set.MembTest(x, SetSize, ChunkSize) {
								hitSetId = i
								break
							}
						}
					}
					results <- hitSetId
				}(startIndex, endIndex)
			}

			wg.Wait()
			close(results)

			for i := range results {
				if hitSetId > i {
					hitSetId = i
				}
			}

			// found hit set
			if hitSetId != 999999999 {
				break
			}
		}
		*/

		var xVal util.DBEntry

		// if still no hit set found, then fail
		if hitSetId == 999999999 {
			if v, ok := localMissElements[x]; ok == false {
				log.Fatalf("No hit set found for %v in %v-th query", x, q)
			} else {
				log.Printf("Hit missing and cached element %v", x)
				xVal = v
				randSet := make([]uint64, SetSize)
				for i := uint64(0); i < SetSize; i++ {
					randSet[i] = rng.Uint64()%ChunkSize + i*ChunkSize
				}
				// send the dummy set to the server
				_, err := leftClient.SetParityQuery(ctx, &pb.SetParityQueryMsg{SetSize: SetSize, Indices: randSet})
				if err != nil {
					log.Fatalf("failed to make punct set query to server %v", err)
				}
				localCache[x] = xVal
				continue
			}
		}

		/************ Now we have a hit set. ************/

		// expand the set
		set := util.PRSetWithShortTag{Tag: localSets[hitSetId].tag}
		//expandedSet := set.Expand(&masterKey, SetSize, ChunkSize)
		expandedSet := set.ExpandWithLongKey(longKey, SetSize, ChunkSize)

		// manually program the set if the flag is set before
		if localSets[hitSetId].isProgrammed {
			programmedChunkId := localSets[hitSetId].programmedPoint / ChunkSize
			expandedSet[programmedChunkId] = localSets[hitSetId].programmedPoint
		}

		// edit the set by replacing the chunk(x)-th element with a replacement
		nxtAvailable := LocalReplacementGroups[chunkId].consumed
		repIndex := rng.Uint64()%ChunkSize + chunkId*ChunkSize // a dummy random index
		repVal := util.ZeroEntry()
		if nxtAvailable == backupSetNumPerGroup {
			log.Printf("No replacement available for %v-th query", q)
		} else {
			// consume one replacement
			repIndex = LocalReplacementGroups[chunkId].indices[nxtAvailable]
			repVal = LocalReplacementGroups[chunkId].value[nxtAvailable]
			LocalReplacementGroups[chunkId].consumed++
		}
		expandedSet[chunkId] = repIndex

		/************ Now send the edited set to the server. ************/
		networkStart := time.Now()
		res, err := leftClient.SetParityQuery(ctx, &pb.SetParityQueryMsg{SetSize: SetSize, Indices: expandedSet})
		networkSince := time.Since(networkStart)
		if err != nil {
			log.Fatalf("failed to make punct set query to server %v", err)
		}

		remoteTotalTime := networkSince.Nanoseconds()
		networkLatency := uint64(remoteTotalTime) - res.ServerComputeTime
		totalNetworkLatency += networkLatency
		totalServerComputeTime += res.ServerComputeTime

		/*********** Now the res.Parity contains the parity of the edited set. ***********/
		/*********** We can recover the answer now. ***********/

		xVal = localSets[hitSetId].parity                              // the parity of the hit set
		util.DBEntryXorFromRaw(&xVal, res.Parity[:util.DBEntryLength]) // xor the parity of the edited set
		util.DBEntryXor(&xVal, &repVal)                                // xor the replacement value

		// update the local cache
		localCache[x] = xVal

		// verify the correctness of the query
		entry := util.GenDBEntry(DBSeed, x)
		// if ignoreOffline == true, the client will not verify the correctness of the query
		if ignoreOffline == false && util.EntryIsEqual(&xVal, &entry) == false {
			log.Fatalf("wrong value %v at index %v at query %v", xVal, x, q)
		} else {
			if q == 0 {
				log.Printf("Correct value %v at index %v", xVal, x)
			}
		}

		// Regresh Phase:
		// The client picks one set from the x's backup group
		// adds the x to the set and adds the set to the local set list

		if LocalBackupSetGroups[chunkId].consumed == backupSetNumPerGroup {
			log.Printf("consumed %v sets", LocalBackupSetGroups[chunkId].consumed)
			log.Printf("backupSetNumPerGroup %v", backupSetNumPerGroup)
			log.Fatalf("No backup set available for %v-th query", q)
		}

		consumed := LocalBackupSetGroups[chunkId].consumed
		localSets[hitSetId].tag = LocalBackupSetGroups[chunkId].sets[consumed].tag
		util.DBEntryXor(&xVal, &LocalBackupSetGroups[chunkId].sets[consumed].parityAfterPunct)
		localSets[hitSetId].parity = xVal
		localSets[hitSetId].isProgrammed = true
		localSets[hitSetId].programmedPoint = x
		LocalBackupSetGroups[chunkId].consumed++
	}

	elapsed = time.Since(start)
	perQueryUploadCost := float64(SetSize) * float64(8)
	perQueryDownloadCost := float64(uint64(reflect.TypeOf(util.DBEntry{}).Size()))

	avgNetworkLatency := float64(totalNetworkLatency) / float64(totalQueryNum)
	avgServerComputeTime := float64(totalServerComputeTime) / float64(totalQueryNum)
	avgAmortizedTime := float64(elapsed.Nanoseconds()) / float64(totalQueryNum)
	avgClientComputeTime := avgAmortizedTime - avgNetworkLatency - avgServerComputeTime
	avgFindHintTime := float64(totalFindHintTime) / float64(totalQueryNum)

	log.Printf("Finish Online Phase with %v queries", totalQueryNum)
	log.Printf("Online Phase took %v ms, amortized time %v ms", elapsed.Milliseconds(), float64(elapsed.Milliseconds())/float64(totalQueryNum))
	log.Printf("Per query upload cost %v kb", perQueryUploadCost/1024)
	log.Printf("Per query download cost %v kb", perQueryDownloadCost/1024)
	log.Printf("End to end amortized time %v ms", float64(offlineElapsed.Milliseconds())/float64(plannedQueryNum)+float64(elapsed.Milliseconds())/float64(totalQueryNum))
	log.Printf("End to end amortized comm cost %v kb", (float64(offlineCommCost)/1024/float64(plannedQueryNum) + (perQueryUploadCost+perQueryDownloadCost)/1024))

	log.Printf("---------------breakdown-------------------------")
	log.Printf("End to end amortized time %v ms", float64(offlineElapsed.Milliseconds())/float64(plannedQueryNum)+float64(elapsed.Milliseconds())/float64(totalQueryNum))
	log.Printf("Average Online Time %v ms", avgAmortizedTime/1000000)
	log.Printf("Average Network Latency %v ms", avgNetworkLatency/1000000)
	log.Printf("Average Server Time %v ms", avgServerComputeTime/1000000)
	log.Printf("Average Client Time %v ms", avgClientComputeTime/1000000)
	log.Printf("Average Find Hint Time %v ms", avgFindHintTime/1000000)
	log.Printf("-------------------------------------------------")

	str = fmt.Sprintf("Finish Online Phase with %v queries\n", totalQueryNum)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Online Phase took %v ms, amortized time %v ms\n", elapsed.Milliseconds(), float64(elapsed.Milliseconds())/float64(totalQueryNum))
	LogFile.WriteString(str)

	str = fmt.Sprintf("Per query upload cost %v kb\n", float64(perQueryUploadCost)/1024)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Per query download cost %v kb\n", float64(perQueryDownloadCost)/1024)
	LogFile.WriteString(str)

	str = fmt.Sprintf("End to end amortized time %v ms", float64(offlineElapsed.Milliseconds())/float64(plannedQueryNum)+float64(elapsed.Milliseconds())/float64(totalQueryNum))
	LogFile.WriteString(str)

	str = fmt.Sprintf("End to end amortized comm cost %v kb", (float64(offlineCommCost)/1024/float64(plannedQueryNum) + (perQueryUploadCost+perQueryDownloadCost)/1024))
	LogFile.WriteString(str)

	str = fmt.Sprintf("---------------breakdown-------------------------")
	LogFile.WriteString(str)

	str = fmt.Sprintf("End to end amortized time %v ms", float64(offlineElapsed.Milliseconds())/float64(plannedQueryNum)+float64(elapsed.Milliseconds())/float64(totalQueryNum))
	LogFile.WriteString(str)

	str = fmt.Sprintf("Average Online Time %v ms", avgAmortizedTime/1000000)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Average Network Latency %v ms", avgNetworkLatency/1000000)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Average Server Time %v ms", avgServerComputeTime/1000000)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Average Client Time %v ms", avgClientComputeTime/1000000)
	LogFile.WriteString(str)

	str = fmt.Sprintf("Average Find Hint Time %v ms", avgFindHintTime/1000000)
	LogFile.WriteString(str)

	str = fmt.Sprintf("-------------------------------------------------")
	LogFile.WriteString(str)
}

func main() {
	addrPtr := flag.String("ip", "localhost:50051", "port number")
	threadPtr := flag.Int("thread", 1, "number of threads")
	ignoreOfflinePtr := flag.Bool("ignoreOffline", false, "ignore offline phase")
	flag.Parse()

	serverAddr = *addrPtr
	threadNum = uint64(*threadPtr)
	ignoreOffline = *ignoreOfflinePtr
	log.Printf("Server address %v, thread number %v ignoreOffline %v", serverAddr, threadNum, ignoreOffline)

	DBSize, DBSeed = ReadConfigInfo()
	ChunkSize, SetSize = util.GenParams(DBSize)

	maxMsgSize := 12 * 1024 * 1024

	f, _ := os.OpenFile("output.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	LogFile = f

	log.Printf("DBSize %v, DBSeed %v, ChunkSize %v, SetSize %v", DBSize, DBSeed, ChunkSize, SetSize)

	leftConn, err := grpc.Dial(
		serverAddr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)),
	)
	if err != nil {
		log.Fatalf("Failed to connect server %v", leftAddress)
	}
	leftClient := pb.NewQueryServiceClient(leftConn)

	defer leftConn.Close()

	runPIRWithOneServer(leftClient, DBSize, DBSeed)
}
