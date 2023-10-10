package test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "scmc/rpc/pb/image"
)

func TestImageList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewImageClient(conn)
		request := pb.ListRequest{
			NodeId: 1,
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		for _, i := range reply.Images {
			t.Logf("%+v", i)
		}

	})
}

func TestImageListDB(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewImageClient(conn)
		request := pb.ListDBRequest{}

		reply, err := cli.ListDB(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		for _, i := range reply.Images {
			t.Logf("%+v", i)
		}

		t.Logf("List reply: %v", reply)
	})
}

func GetHash(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("open file %v err: %v", filename, err)
		return "", err
	}
	defer file.Close()

	hash256 := sha256.New()
	_, err = io.Copy(hash256, file)
	if err != nil {
		fmt.Printf("io copy error: %v", err)
		return "", err
	}

	hashStr := hex.EncodeToString(hash256.Sum(nil))

	return hashStr, nil
}

func TestImageUpload(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewImageClient(conn)
		t.Parallel()
		imagePath := "/root/tmp/hello-world-0221.tar"
		file, err := os.Open(imagePath)
		require.NoError(t, err)
		defer file.Close()

		stream, err := cli.Upload(context.Background())
		require.NoError(t, err)
		imageType := filepath.Ext(imagePath)
		fileStat, err := os.Stat(imagePath)
		require.NoError(t, err)
		hashStr, err := GetHash(imagePath)
		require.NoError(t, err)

		req := &pb.UploadRequest{
			Info: &pb.UploadInfo{
				Name:     "hello-world-0221",
				Version:  "v1.0",
				Type:     imageType,
				Checksum: hashStr,
				Size:     fileStat.Size(),
			},
		}

		err = stream.Send(req)
		require.NoError(t, err)

		reader := bufio.NewReader(file)
		buffer := make([]byte, 1024*1024)
		size := 0

		for {
			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}

			require.NoError(t, err)
			size += n

			req := &pb.UploadRequest{
				ChunkData: buffer[:n],
			}

			err = stream.Send(req)
			require.NoError(t, err)
		}

		_, err = stream.CloseAndRecv()
		require.NoError(t, err)

	})
}

func TestImageUpdate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewImageClient(conn)
		t.Parallel()
		imagePath := "/root/python/hello-world-python.tar"
		file, err := os.Open(imagePath)
		require.NoError(t, err)
		defer file.Close()

		stream, err := cli.Update(context.Background())
		require.NoError(t, err)
		imageType := filepath.Ext(imagePath)
		fileStat, err := os.Stat(imagePath)
		require.NoError(t, err)
		hashStr, err := GetHash(imagePath)
		require.NoError(t, err)

		req := &pb.UpdateRequest{
			ImageId: 3,
			Info: &pb.UploadInfo{
				Name:     "hello-world-python",
				Version:  "v3.1",
				Type:     imageType,
				Checksum: hashStr,
				Size:     fileStat.Size(),
			},
		}

		err = stream.Send(req)
		require.NoError(t, err)

		reader := bufio.NewReader(file)
		buffer := make([]byte, 1024*1024)
		size := 0

		for {
			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}

			require.NoError(t, err)
			size += n

			req := &pb.UpdateRequest{
				ChunkData: buffer[:n],
			}

			err = stream.Send(req)
			require.NoError(t, err)
		}

		_, err = stream.CloseAndRecv()
		require.NoError(t, err)
	})
}

func TestImageDownload(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewImageClient(conn)
		request := pb.DownloadRequest{
			ImageId: int64(1),
		}

		stream, err := cli.Download(ctx, &request)
		if err != nil {
			t.Errorf("download: %v", err)
		}

		reply, err := stream.Recv()
		if err != nil {
			t.Errorf("cannot receive image info %v", err)
		}

		if reply.Info == nil {
			t.Errorf("cannot reply info is nil")
		}

		imageData := bytes.Buffer{}
		var imageSize int64
		fileName := fmt.Sprintf("/root/tmp/file_transfer/dst/%s_%s%s", reply.Info.Name, reply.Info.Version, reply.Info.Type)
		t.Logf("test write file %v", fileName)
		file, err := os.Create(fileName)
		if err != nil {
			t.Errorf("cannot create file: %v", fileName)
		}
		defer file.Close()

		for {
			if stream.Context().Err() == context.Canceled || stream.Context().Err() == context.DeadlineExceeded {
				t.Errorf("context error: %v", stream.Context().Err())
			}

			reply, err := stream.Recv()
			if err == io.EOF {
				t.Logf("no more data")
				break
			}
			if err != nil {
				t.Errorf("cannot receive chunk data: %v", err)
			}

			chunk := reply.ChunkData
			size := len(chunk)

			imageSize += int64(size)
			if imageSize > (1<<30) || imageSize > reply.Info.Size {
				t.Errorf("image is too large: [%v] > [1 << 30] || [%v]", imageSize, reply.Info.Size)
			}

			// write slowly
			time.Sleep(time.Millisecond)

			_, err = imageData.Write(chunk)
			if err != nil {
				t.Errorf("cannot write chunk data: %v", err)
			}

			_, err = file.Write(chunk)
			if err != nil {
				t.Errorf("cannot write chunk data to file: %v", err)
			}
		}

		hashStr, err := GetHash(fileName)
		require.NoError(t, err)
		if hashStr != reply.Info.Checksum {
			t.Errorf("[%v] != [%v]", hashStr, reply.Info.Checksum)
		}
		t.Logf("download end")
	})
}

func TestImageApprove(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewImageClient(conn)
		request := pb.ApproveRequest{
			ImageId:      int64(2),
			Approve:      true,
			RejectReason: "",
		}

		reply, err := cli.Approve(ctx, &request)
		if err != nil {
			t.Errorf("Approve: %v", err)
		}

		t.Logf("Approve reply: %v", reply)
	})
}

func TestImageRemove(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewImageClient(conn)
		request := pb.RemoveRequest{
			ImageIds: []int64{1},
		}

		reply, err := cli.Remove(ctx, &request)
		if err != nil {
			t.Errorf("Remove: %v", err)
		}

		t.Logf("Remove reply: %v", reply)
	})
}
