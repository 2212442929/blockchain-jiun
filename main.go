package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type JiunBlock struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
}

type Message struct {
	BPM int
}

var Blockchain []JiunBlock

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		JiunBlock := JiunBlock{0, t.String(), 0, "", ""}
		spew.Dump(JiunBlock)
		Blockchain = append(Blockchain, JiunBlock)
	}()
	log.Fatal(run())
}

func calculateHash(JiunBlock JiunBlock) string {
	sum := string(rune(JiunBlock.Index)) + string(rune(JiunBlock.BPM)) + JiunBlock.Timestamp + JiunBlock.PrevHash
	hash := sha256.New()
	hash.Write([]byte(sum))
	return hex.EncodeToString(hash.Sum(nil))
}

func generateBlock(oldBlock JiunBlock, BPM int) (JiunBlock, error) {
	var newBlock JiunBlock
	newBlock.Index = oldBlock.Index + 1
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Timestamp = time.Now().String()
	newBlock.Hash = calculateHash(newBlock)
	return newBlock, nil
}

func checkBlock(newBlock, oldBlock JiunBlock) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if newBlock.PrevHash != oldBlock.Hash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

func replaceChain(newBlocks []JiunBlock) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("PORT")
	log.Println("Listen on ", httpAddr)
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func makeMuxRouter() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/", handleGetBlockchain).Methods("GET")
	router.HandleFunc("/", handleWriteBlock).Methods("POST")
	return router
}

func handleWriteBlock(writer http.ResponseWriter, request *http.Request) {
	var m Message
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(writer, request, http.StatusBadRequest, request.Body)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(request.Body)

	JiunBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
	if err != nil {
		respondWithJSON(writer, request, http.StatusInternalServerError, m)
		return
	}
	if checkBlock(JiunBlock, Blockchain[len(Blockchain)-1]) {
		newBlockchain := append(Blockchain, JiunBlock)
		replaceChain(newBlockchain)
		spew.Dump(Blockchain)
	}

	respondWithJSON(writer, request, http.StatusCreated, Blockchain)

}

func respondWithJSON(writer http.ResponseWriter, request *http.Request, created int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, err := writer.Write([]byte("HTTP 500: Internal Server Error"))
		if err != nil {
			return
		}
		return
	}
	writer.WriteHeader(created)
	_, err = writer.Write(response)
	if err != nil {
		return
	}
}

func handleGetBlockchain(writer http.ResponseWriter, request *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", " ")
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.WriteString(writer, string(bytes))
	if err != nil {
		return
	}
}
