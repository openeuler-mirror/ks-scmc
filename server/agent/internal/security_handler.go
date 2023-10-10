package internal

import (
	"archive/tar"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"scmc/common"
	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/security"
)

type SecurityServer struct {
	pb.UnimplementedSecurityServer
	flag      bool
	flagGuard sync.Mutex
}

func (s *SecurityServer) UpdateFileProtection(ctx context.Context, in *pb.UpdateFileProtectionRequest) (*pb.UpdateFileProtectionReply, error) {
	reply := pb.UpdateFileProtectionReply{}
	if err := model.UpdateFileAccess(in.ContainerId, in.IsOn, in.ToAppend, in.ToRemove); err != nil {
		return &reply, err
	}

	return &reply, nil
}

func (s *SecurityServer) LoadSecurityConfig(ctx context.Context, in *pb.LoadSecurityConfigRequset) (*pb.LoadSecurityConfigReply, error) {
	reply := pb.LoadSecurityConfigReply{}
	s.flagGuard.Lock()
	defer s.flagGuard.Unlock()
	if !s.flag {
		log.Debugf("resume security config")
		for _, v := range in.Configs {
			if v.FileProtections != nil {
				log.Debugf("resume file access: [%v], [%v]", v.ContainerId, v.FileProtections)
				if err := model.UpdateFileAccess(v.ContainerId, v.FileProtections.IsOn, v.FileProtections.FileList, []string{}); err != nil {
					log.Warnf("UpdateFileAccess %v err: %v", v.ContainerId, err)
				}
			}

			if v.ProcProtections != nil {
				log.Debugf("resume white list: [%v], [%v]", v.ContainerId, v.ProcProtections)
				if err := model.UpdateWhiteList(v.ContainerId, v.ProcProtections.IsOn, v.ProcProtections.ExeList, []string{}); err != nil {
					log.Warnf("UpdateFileAccess %v err: %v", v.ContainerId, err)
				}
			}
		}

		s.flag = true
	}

	return &reply, nil
}

func (sp *SecurityServer) ListProcProtection(ctx context.Context, in *pb.ListProcProtectionRequest) (*pb.ListProcProtectionReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	path := common.Config.Agent.OpensnitchRuleDir

	rule, err := model.LoadOpensnitchRule(path, in.ContainerId)
	if err != nil {
		return nil, err
	}
	if rule.Operator.Type != "list" || rule.Operator.Operand != "list" {
		return nil, rpc.ErrInternal
	}
	var opData string
	for _, op := range rule.Operator.List {
		if op.Operand == "process.path" {
			opData = op.Data
		}
	}

	if opData[:2] == "^(" {
		opData = opData[2:]
	}

	if opData[len(opData)-2:] == ")$" {
		opData = opData[:len(opData)-2]
	}

	exeList := strings.Split(opData, "|")
	var reply *pb.ListProcProtectionReply = &pb.ListProcProtectionReply{
		IsOn:    rule.Enabled,
		ExeList: exeList,
	}

	return reply, nil
}

func calcdockerCPMD5(ctx context.Context, containerId, srcDir string) (string, error) {
	cli, err := model.DockerClient()
	if err != nil {
		return "", rpc.ErrInternal
	}

	content, stat, err := cli.CopyFromContainer(ctx, containerId, srcDir)
	if err != nil {
		return "", err
	}
	defer content.Close()
	if stat.Mode.IsDir() {
		return "", fmt.Errorf("is dir")
	}

	tarContent := tar.NewReader(content)
	md5Handle := md5.New()
	for {
		// 读取每一块内容
		_, err := tarContent.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		_, err = io.Copy(md5Handle, tarContent)
		if nil != err {
			return "", err
		}
	}

	md5Str := hex.EncodeToString(md5Handle.Sum(nil))
	fmt.Println(md5Str)
	return md5Str, nil
}

func generateMD5Slice(containerId string, in []string) (out []string, err error) {
	if len(in) == 0 {
		return out, nil
	}

	ctx := context.Background()
	for _, path := range in {
		md5, err := calcdockerCPMD5(ctx, containerId, path)
		if err != nil {
			log.Warnf("calcFileMD5: %v", err)
			return nil, rpc.ErrInvalidArgument
		}
		log.Debugf("md5: %s", md5)
		out = append(out, md5)
	}
	return out, nil
}

func (sp *SecurityServer) UpdateProcProtection(ctx context.Context, in *pb.UpdateProcProtectionRequest) (*pb.UpdateProcProtectionReply, error) {
	if in.NodeId <= 0 || in.ContainerId == "" {
		return nil, rpc.ErrInvalidArgument
	}
	switch in.ProtectionType {
	case int64(pb.PROC_PROTECTION_EXEC_WHITELIST):
		log.Debugln("update  process protection")
		if !in.IsOn {
			if err := model.CleanWhiteList(in.ContainerId); err != nil {
				log.Warnf("delete %s process protection file faild,err is :%s", in.ContainerId, err)
				return nil, err
			}
		} else {
			toAppend, err := generateMD5Slice(in.ContainerId, in.ToAppend)
			if err != nil {
				return nil, err
			}
			toRemove, err := generateMD5Slice(in.ContainerId, in.ToRemove)
			if err != nil {
				return nil, err
			}
			if err := model.UpdateWhiteList(in.ContainerId, in.IsOn, toAppend, toRemove); err != nil {
				log.Warnf("add %s network process protection file faild,err is :%s", in.ContainerId, err)
				return nil, err
			}
		}
	case int64(pb.PROC_PROTECTION_NET_WHITELIST):
		//更新节点上规则
		log.Debugln("update network process protection")
		if !in.IsOn {
			if err := model.RemoveOpensnitchRule(in.ContainerId); err != nil {
				log.Warnf("delete %s network process protection file faild,err is :%s", in.ContainerId, err)
				return nil, err
			}
		} else {
			var inrule = &model.ProcProtection{
				Type:    int32(in.ProtectionType),
				IsOn:    true,
				ExeList: in.ToAppend,
			}
			if err := model.SaveOpensnitchRule(inrule, in.ContainerId, "uuid"); err != nil {
				log.Warnf("add %s network process protection file faild,err is :%s", in.ContainerId, err)
				return nil, err
			}
		}

	default:
		return nil, rpc.ErrInvalidArgument
	}

	return &pb.UpdateProcProtectionReply{}, nil
}
