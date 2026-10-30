package common

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"google.golang.org/grpc/credentials"
)

/*
 * gRPC 走 https 需要用到的
 */
func LoadTLSCredentials() (credentials.TransportCredentials, error) {
	if !Config.TLS.Enable {
		return nil, nil
	}

	if Config.TLS.CA == "" || Config.TLS.ServerCert == "" || Config.TLS.ServerKey == "" {
		return nil, nil
	}

	caCert, err := ioutil.ReadFile(Config.TLS.CA)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA certificate")
	}

	serverCert, err := tls.LoadX509KeyPair(Config.TLS.ServerCert, Config.TLS.ServerKey)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    certPool,
	}), nil
}

/*
 * thrift 走证书双向认证的时候需要用到的 
 */
func LoadTLSConfig() (*tls.Config, error) {
	if Config.TLS.CA == "" || Config.TLS.ServerCert == "" || Config.TLS.ServerKey == "" {
		return nil, fmt.Errorf("not find ca, serverCert, serverCert!")
	}

	caCert, err := ioutil.ReadFile(Config.TLS.CA)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA certificate")
	}

	serverCert, err := tls.LoadX509KeyPair(Config.TLS.ServerCert, Config.TLS.ServerKey)
	if err != nil {
		return nil, err
	}

        // 验证 客户端证书， 强制性质的 
	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}, nil
}

/*
 * thrift 走证书双向认证的时候需要用到的 
 */
func LoadApiTLSConfig() (*tls.Config, error) {
	if Config.ApiTLS.CA == "" || Config.ApiTLS.ServerCert == "" || Config.ApiTLS.ServerKey == "" {
		return nil, fmt.Errorf("not find ca, serverCert, serverCert!")
	}

	caCert, err := ioutil.ReadFile(Config.ApiTLS.CA)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA certificate")
	}

	serverCert, err := tls.LoadX509KeyPair(Config.ApiTLS.ServerCert, Config.ApiTLS.ServerKey)
	if err != nil {
		return nil, err
	}

        // 验证 客户端证书， 强制性质的 
	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}, nil
}
