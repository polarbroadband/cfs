package main

import (
	"context"
	"fmt"
	"os"

	pb "github.com/polarbroadband/cfs/pkg/cfsprotobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	log "github.com/sirupsen/logrus"
)

var (
	// container image release
	RELEASE = os.Getenv("RELEASE_CLNT")
	// container name
	HOST = os.Getenv("HOST_CLNT")
	CFS  = os.Getenv("HOST_CFS")
	// JWT shared secret
	TOKENSEC = []byte(os.Getenv("BACKEND_TOKEN"))
)

func init() {
	// config package level default logger
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.TraceLevel)
}

func main() {

	hold := make(chan struct{})

	caCer, err := credentials.NewClientTLSFromFile("/cert/ca.cer", "")
	if err != nil {
		log.Panicf("unable to import ca certificate: %v", err)
	}
	// Set up TLS connection to the server
	cfsConn, err := grpc.Dial(CFS+":50051", grpc.WithTransportCredentials(caCer))
	if err != nil {
		log.Panic("unable to connect %s: %v", CFS, err)
	}
	defer cfsConn.Close()
	cfs := pb.NewCfsClient(cfsConn)
	gCtx, gCancel := context.WithCancel(context.Background())
	defer gCancel()
	if res, err := cfs.Healtz(gCtx, &pb.HealtzReq{}); err != nil {
		log.Panicf("%s gRPC healtz check fail: %v", CFS, err)
	} else {
		fmt.Printf("\n*** gRPC healtz check ***\nHost: %s\nRel: %s\n", res.GetHost(), res.GetRelease())
	}

	if cs, err := cfs.CheckFile(gCtx, &pb.CheckFileRequest{FilePath: "gear", FileName: "Regression_Service.tcc"}); err != nil {
		log.Panicf("file check fail: %v", err)
	} else {
		fmt.Printf("\n*** CheckSum ***\n%s\n", cs.GetChkSum())
	}
	/*
		if cs, err := cfs.LoadSpirentTC(gCtx, &pb.LoadSpirentFileRequest{
			FilePath:   "gear",
			FileName:   "Regression_Service.tcc",
			SpirentURL: "http://172.18.160.44/stcapi/files",
			SessionID:  "t11p - x229370",
		}); err != nil {
			log.Panicf("file upload fail: %v", err)
		} else {
			fmt.Printf("\n*** CheckSum ***\n%s\n", cs.GetChkSum())
		}
	*/
	if cs, err := cfs.SCP(gCtx, &pb.SCPFileRequest{
		FilePath: "gear",
		//FileName: "Regression_Service.tcc",
		FileName: "ca.cert",
		Host:     "172.18.132.185", //sros
		// Host:     "172.18.132.79",	//junos
		//Host: "172.18.160.53",	//linux
		// RemoteFilePath: `/var/home/remote`,	// junos
		RemoteFilePath: `/image`, // sros
		//RemoteFilePath: "/home/t854359/projects/netsight/jj",
		Usr: "t854359",
		Pwd: "k1337388O",
		//Pwd: "lab2020",
	}); err != nil {
		log.Panicf("file transfer fail: %v", err)
	} else {
		fmt.Printf("\n*** CheckSum ***\n%s\n", cs.GetChkSum())
	}

	<-hold
}
