package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	pb "github.com/polarbroadband/cfs/pkg/cfsprotobuf"

	"github.com/polarbroadband/goto/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"

	log "github.com/sirupsen/logrus"
)

var (
	// container image release
	RELEASE = os.Getenv("RELEASE_CFS")
	// container name
	HOST = os.Getenv("HOST_CFS")
	// JWT shared secret
	TOKENSEC = []byte(os.Getenv("BACKEND_TOKEN"))

	// root path of file storage
	ROOT = os.Getenv("CFS_ROOT") + "/"
)

func init() {
	// config package level default logger
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.TraceLevel)
}

type WorkerNode struct {
	pb.UnimplementedCfsServer
	*util.API
}

func main() {
	wkr := WorkerNode{
		API: &util.API{
			TokenSec: TOKENSEC,
			NoAuth: []string{
				"/cfsprotobuf.cfs/Healtz",
				"/cfsprotobuf.cfs/LoadSpirentTC",
				"/cfsprotobuf.cfs/SCP",
				"/cfsprotobuf.cfs/CheckFile",
			},
			Log: log.WithField("owner", HOST),
		},
	}

	// setup and run gRPC server
	grpcTLS, err := credentials.NewServerTLSFromFile("/cert/cfs.cer", "/cert/cfs.key")
	if err != nil {
		wkr.Log.WithError(err).Fatal("gRPC server fail: invalid TLS keys")
	}
	grpcSvr := grpc.NewServer(grpc.Creds(grpcTLS), grpc.UnaryInterceptor(wkr.AuthGrpcUnary), grpc.StreamInterceptor(wkr.AuthGrpcStream))
	pb.RegisterCfsServer(grpcSvr, &wkr)
	// start probe grpc server
	grpcListener, err := net.Listen("tcp", ":50051")
	if err != nil {
		wkr.Log.WithError(err).Fatal("gRPC server fail: unable to init tcp socket 50051")
	}
	go func() {
		wkr.Log.Info("gRPC server start")
		wkr.Log.Fatal(grpcSvr.Serve(grpcListener))
	}()

	hold := make(chan struct{})
	<-hold
}

// Healtz response gRPC health check
func (wkr *WorkerNode) Healtz(ctx context.Context, r *pb.HealtzReq) (*pb.SvrStat, error) {
	//_e := util.NewExeErr("Healtz", HOST, "gRPC_API")
	return &pb.SvrStat{
		Host:    HOST,
		Release: RELEASE,
	}, nil
}

func (wkr *WorkerNode) LoadSpirentTC(ctx context.Context, r *pb.LoadSpirentFileRequest) (*pb.FileCheckSum, error) {
	_e := util.NewExeErr("LoadSpirentTC", HOST, "gRPC_API")
	file := ROOT + r.GetFilePath() + "/" + r.GetFileName()
	err, _, chksum := util.FileExist(file, "")
	if err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("file not available "+file, err))
	}
	fileHdl, err := os.Open(file)
	if err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("unable to open file "+file, err))
	}
	defer fileHdl.Close()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // test server certificate is not trusted.
			},
		},
	}
	req, err := http.NewRequest("POST", r.GetSpirentURL(), fileHdl)
	if err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("unable to build http POST request "+r.GetSpirentURL(), err))
	}
	req.Header.Set("X-STC-API-Session", r.GetSessionID())
	req.Header.Set("Content-Type", "binary/octet-stream")
	req.Header.Set("content-disposition", "attachment; filename="+file)

	wkr.Log.Tracef("API POST endpoint: %s", r.GetSpirentURL())
	resp, err := client.Do(req)
	if err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("failed request for "+r.GetSpirentURL(), err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("failed request for "+r.GetSpirentURL(), resp.Status))
	}

	return &pb.FileCheckSum{ChkSum: chksum}, nil
}

func (wkr *WorkerNode) SCP(ctx context.Context, r *pb.SCPFileRequest) (*pb.FileCheckSum, error) {
	_e := util.NewExeErr("SCP", HOST, "gRPC_API")
	file := ROOT + r.GetFilePath() + "/" + r.GetFileName()
	err, _, chksum := util.FileExist(file, "")
	if err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("file not available "+file, err))
	}

	sshConfig := &ssh.ClientConfig{
		User:            r.GetUsr(),
		Auth:            []ssh.AuthMethod{ssh.Password(r.GetPwd())},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
		Config:          ssh.Config{ /*
				KeyExchanges: []string{
					"diffie-hellman-group1-sha1",
					"diffie-hellman-group14-sha1",
				},*/
		},
	}
	sshClient, err := ssh.Dial("tcp", r.GetHost()+":22", sshConfig)
	if err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("unable to ssh host "+r.GetHost(), err))
	}
	defer sshClient.Close()

	sessionSCP, err := sshClient.NewSession()
	if err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("unable to setup session to host "+r.GetHost(), err))
	}
	defer sessionSCP.Close()

	if err = scp.CopyPath(file, r.GetRemoteFilePath()+"/"+r.GetFileName(), sessionSCP); err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String(fmt.Sprintf("fail to scp file %s to %s", file, r.GetHost()), err))
	}
	return &pb.FileCheckSum{ChkSum: chksum}, nil
}

func (wkr *WorkerNode) CheckFile(ctx context.Context, r *pb.CheckFileRequest) (*pb.FileCheckSum, error) {
	_e := util.NewExeErr("CheckFile", HOST, "gRPC_API")
	file := ROOT + r.GetFilePath() + "/" + r.GetFileName()
	err, _, chksum := util.FileExist(file, r.GetChkSum())
	if err != nil {
		return nil, wkr.Errpc(codes.Unavailable, _e.String("file not available "+file, err))
	}
	return &pb.FileCheckSum{ChkSum: chksum}, nil
}
