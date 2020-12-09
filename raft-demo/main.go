package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/raft"

	"cloudfreexiao/raft-demo/fsm"
	"cloudfreexiao/raft-demo/mraft"
)

var (
	httpAddr    string
	raftAddr    string
	raftID      string
	raftCluster string
	raftDir     string
)

var (
	isLeader bool
)

func init() {
	flag.StringVar(&httpAddr, "http_addr", "127.0.0.1:7001", "http listen addr")
	flag.StringVar(&raftAddr, "raft_addr", "127.0.0.1:7000", "raft listen addr")
	flag.StringVar(&raftID, "raft_id", "1", "raft id")
	flag.StringVar(&raftCluster, "raft_cluster", "1/127.0.0.1:7000,2/127.0.0.1:8000,3/127.0.0.1:9000", "cluster info")
}

func main() {
	flag.Parse()
	// 初始化配置
	if httpAddr == "" || raftAddr == "" || raftID == "" || raftCluster == "" {
		fmt.Println("config error")
		os.Exit(1)
		return
	}
	raftDir := "node/raft_" + raftID
	os.MkdirAll(raftDir, 0700)

	// 初始化 raft
	mRaft, fm, err := mraft.NewMRaft(raftAddr, raftID, raftDir)
	if err != nil {
		fmt.Println("NewMyRaft error ", err)
		os.Exit(1)
		return
	}

	// 启动 raft
	mraft.Bootstrap(mRaft, raftID, raftAddr, raftCluster)

	// 监听leader变化
	go func() {
		for leader := range mRaft.LeaderCh() {
			isLeader = leader
		}
	}()

	// 启动http server
	httpServer := HTTPServer{
		ctx: mRaft,
		fsm: fm,
	}

	http.HandleFunc("/set", httpServer.Set)
	http.HandleFunc("/get", httpServer.Get)
	http.ListenAndServe(httpAddr, nil)

	// 关闭raft
	shutdownFuture := mRaft.Shutdown()
	if err := shutdownFuture.Error(); err != nil {
		fmt.Printf("shutdown raft error:%v \n", err)
	}

	// 退出http server
	fmt.Println("shutdown kv http server")
}

type HTTPServer struct {
	ctx *raft.Raft
	fsm *fsm.Fsm
}

func (h HTTPServer) Set(w http.ResponseWriter, r *http.Request) {
	if !isLeader {
		fmt.Fprintf(w, "not leader")
		return
	}
	vars := r.URL.Query()
	key := vars.Get("key")
	value := vars.Get("value")
	if key == "" || value == "" {
		fmt.Fprintf(w, "error key or value")
		return
	}

	data := "set" + "," + key + "," + value
	future := h.ctx.Apply([]byte(data), 5*time.Second)
	if err := future.Error(); err != nil {
		fmt.Fprintf(w, "error:"+err.Error())
		return
	}
	fmt.Fprintf(w, "ok")
	return
}

func (h HTTPServer) Get(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	key := vars.Get("key")
	if key == "" {
		fmt.Fprintf(w, "error key")
		return
	}
	value := h.fsm.Data[key]
	fmt.Fprintf(w, value)
	return
}
