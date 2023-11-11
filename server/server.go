package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	pb "example.com/query"
	util "example.com/util"
	"google.golang.org/grpc"
)

//const (
//	port = ":50051"
//)

var DBSize uint64
var DBSeed uint64
var ChunkSize uint64
var SetSize uint64
var port string

type QueryServiceServer struct {
	pb.UnimplementedQueryServiceServer
	DB []uint64 // the database, for every DBEntrySize/8 uint64s, we store a DBEntry
}

func (s *QueryServiceServer) DBAccess(id uint64) util.DBEntry {
	if id < uint64(len(s.DB)) {
		if id*util.DBEntryLength+util.DBEntryLength > uint64(len(s.DB)) {
			log.Fatalf("DBAccess: id %d out of range", id)
		}
		return util.DBEntryFromSlice(s.DB[id*util.DBEntryLength : (id+1)*util.DBEntryLength])
	} else {
		var ret util.DBEntry
		for i := 0; i < util.DBEntryLength; i++ {
			ret[i] = 0
		}
		return ret
	}
}

/*
func (s *QueryServiceServer) HandleBatchedQuery(in *pb.BatchedCuckooBucketQuery) *pb.BatchedCuckooBucketResponse {
	num := in.GetQueryNum()
	batchedQuery := in.GetBatchedQuery()
	batchedResponse := make([]*pb.CuckooBucketResponse, num)
	for i := uint64(0); i < num; i++ {
		batchedResponse[i] = s.HandleSingleQuery(batchedQuery[i])
	}

	return &pb.BatchedCuckooBucketResponse{ResponseNum: num, BatchedResponse: batchedResponse}
}
*/

func (s *QueryServiceServer) PlaintextQuery(ctx context.Context, in *pb.PlaintextQueryMsg) (*pb.PlaintextResponse, error) {
	id := in.GetIndex()
	ret := s.DBAccess(id)
	return &pb.PlaintextResponse{Val: ret[:]}, nil
	//return s.HandleSingleQuery(in), nil
	//return &pb.QueryResponse{Value: ret}, nil;
}

func (s *QueryServiceServer) HandleFullSetQuery(key util.PrfKey) util.DBEntry {
	PRSet := util.PRSet{Key: key}
	ExpandedSet := PRSet.Expand(SetSize, ChunkSize)

	var parity util.DBEntry
	for i := 0; i < util.DBEntryLength; i++ {
		parity[i] = 0
	}
	for _, id := range ExpandedSet {
		entry := s.DBAccess(id)
		util.DBEntryXor(&parity, &entry)
	}

	return parity
}

func (s *QueryServiceServer) FullSetQuery(ctx context.Context, in *pb.FullSetQueryMsg) (*pb.FullSetResponse, error) {
	var key util.PrfKey
	copy(key[:], in.GetPRFKey())
	val := s.HandleFullSetQuery(key)
	return &pb.FullSetResponse{Val: val[:]}, nil
}

func (s *QueryServiceServer) HandlePunctSetQuery(indices []uint64) []uint64 {
	guesses := make([]uint64, SetSize*util.DBEntryLength)
	parity := util.ZeroEntry()
	for chunkID, offset := range indices {
		currentID := uint64(chunkID+1)*ChunkSize + offset
		entry := s.DBAccess(currentID)
		util.DBEntryXor(&parity, &entry)
	}

	copy(guesses[0:util.DBEntryLength], parity[:])
	for i := uint64(1); i < SetSize; i++ {
		//  the blank originally is in the (i-1)-th chunk
		// now the blank should be in the i-th chunk
		offset := indices[i-1]
		oldIndex := uint64(i)*ChunkSize + offset
		newIndex := uint64(i-1)*ChunkSize + offset
		entryOld := s.DBAccess(oldIndex)
		entryNew := s.DBAccess(newIndex)
		util.DBEntryXor(&parity, &entryOld)
		util.DBEntryXor(&parity, &entryNew)
		copy(guesses[i*util.DBEntryLength:(i+1)*util.DBEntryLength], parity[:])
	}

	return guesses
}

func (s *QueryServiceServer) PunctSetQuery(ctx context.Context, in *pb.PunctSetQueryMsg) (*pb.PunctSetResponse, error) {
	// start from _, x1, ..., x_{k-1}
	start := time.Now()
	guesses := s.HandlePunctSetQuery(in.GetIndices())
	since := time.Since(start)
	return &pb.PunctSetResponse{ReturnSize: SetSize, ServerComputeTime: uint64(since.Nanoseconds()), Guesses: guesses}, nil
	//return &pb.PunctSetResponse{ReturnSize: SetSize, ServerComputeTime: 0, Guesses: guesses}, nil
}

/*
 * This is the updated query algorithm. Plain and simple.
 * The client sends the indices of the set.
 * The server returns the parity of the set.
 */

func (s *QueryServiceServer) HandleSetParityQuery(indices []uint64) []uint64 {
	parity := util.ZeroEntry()
	for _, index := range indices {
		entry := s.DBAccess(index)
		util.DBEntryXor(&parity, &entry)
	}
	ret := make([]uint64, util.DBEntryLength)
	copy(ret[0:util.DBEntryLength], parity[:])
	return ret
}

func (s *QueryServiceServer) SetParityQuery(ctx context.Context, in *pb.SetParityQueryMsg) (*pb.SetParityQueryResponse, error) {
	// start from _, x1, ..., x_{k-1}
	start := time.Now()
	parity := s.HandleSetParityQuery(in.GetIndices())
	since := time.Since(start)
	return &pb.SetParityQueryResponse{Parity: parity, ServerComputeTime: uint64(since.Nanoseconds())}, nil
}

func (s *QueryServiceServer) BatchedFullSetQuery(ctx context.Context, in *pb.BatchedFullSetQueryMsg) (*pb.BatchedFullSetResponse, error) {
	num := in.GetQueryNum()
	batchedQuery := in.GetQueries()
	batchedResponse := make([]*pb.FullSetResponse, num)
	for i := uint64(0); i < num; i++ {
		var key util.PrfKey
		copy(key[:], batchedQuery[i].GetPRFKey())
		val := s.HandleFullSetQuery(key)
		batchedResponse[i] = &pb.FullSetResponse{Val: val[:]}
	}

	return &pb.BatchedFullSetResponse{ResponseNum: num, Responses: batchedResponse}, nil
}

func (s *QueryServiceServer) FetchFullDB(in *pb.FetchFullDBMsg, stream pb.QueryService_FetchFullDBServer) error {
	for i := uint64(0); i < SetSize; i++ {
		down := i * ChunkSize
		up := (i + 1) * ChunkSize
		var chunk []uint64
		chunk = s.DB[down*util.DBEntryLength : up*util.DBEntryLength]
		/*
			if up > DBSize {
				//fill up with 0
				chunk = s.DB[down*util.DBEntryLength : DBSize*util.DBEntryLength]
				appendSlice := make([]uint64, (up-DBSize)*util.DBEntryLength)
				for j := uint64(0); j < up-DBSize; j++ {
					appendSlice[j] = 0
				}
				chunk = append(chunk, appendSlice...)
			} else {
			}
		*/

		ret := &pb.DBChunk{ChunkId: i, ChunkSize: ChunkSize, Chunk: chunk}
		if err := stream.Send(ret); err != nil {
			log.Printf("Failed to send a chunk: %v", err)
			return err
		}
	}
	return nil
}

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

func main() {
	portPtr := flag.String("port", "50051", "port number")
	ignoreInitPtr := flag.Bool("ignoreInit", false, "ignore initialization of the DB")
	flag.Parse()

	port = *portPtr
	ignoreInit := *ignoreInitPtr
	log.Println("port number: ", port)
	port = ":" + port

	DBSize, DBSeed = ReadConfigInfo()
	log.Printf("DB N: %v, Entry Size %v Bytes, DB Size %v MB", DBSize, util.DBEntrySize, DBSize*util.DBEntrySize/1024/1024)

	ChunkSize, SetSize = util.GenParams(DBSize)

	log.Println("Chunk Size:", ChunkSize, "Set Size:", SetSize)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen %v", err)
	}

	DB := make([]uint64, ChunkSize*SetSize*util.DBEntryLength)
	log.Println("DB Real N:", len(DB))
	if !ignoreInit {
		for i := uint64(0); i < uint64(len(DB))/util.DBEntryLength; i++ {
			//DB[i] = util.DefaultHash(DBSeed ^ i)
			entry := util.GenDBEntry(DBSeed, i)
			copy(DB[i*util.DBEntryLength:(i+1)*util.DBEntryLength], entry[:])
		}
	}
	//for i := DBSize * util.DBEntryLength; i < uint64(len(DB)); i++ {
	//	DB[i] = 0
	//}

	maxMsgSize := 12 * 1024 * 1024

	s := grpc.NewServer(
		grpc.MaxMsgSize(maxMsgSize),
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)

	pb.RegisterQueryServiceServer(s, &QueryServiceServer{DB: DB[:]})
	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to server %v", err)
	}
}
