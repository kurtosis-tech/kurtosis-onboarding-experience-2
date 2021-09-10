package my_advanced_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kurtosis-tech/kurtosis-client/golang/lib/networks"
	"github.com/kurtosis-tech/kurtosis-client/golang/lib/services"
	"github.com/kurtosis-tech/kurtosis-onboarding-experience/smart_contracts/bindings"
	"github.com/kurtosis-tech/kurtosis-testsuite-api-lib/golang/lib/testsuite"
	"github.com/palantir/stacktrace"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
	"time"
)

const (
	lambdaID = "eth-lambda"
	rpcPort                   = 8545

	execCommandSuccessExitCode = 0

	maxNumCheckTransactionMinedRetries      = 10
	timeBetweenCheckTransactionMinedRetries = 1 * time.Second

)

type MyAdvancedTest struct{}

type NodeInfoResponse struct {
	Result NodeInfo `json:"result"`
}

type NodeInfo struct {
	Enode string `json:"enode"`
}

type AddPeerResponse struct {
	Result bool `json:"result"`
}

type EthereumKurtosisLambdaResult struct {
	BootnodeServiceID          services.ServiceID      `json:"bootnode_service_id"`
	NodeServiceIDs             []services.ServiceID    `json:"node_service_ids"`
	StaticFileIDs              []services.StaticFileID `json:"static_file_ids"`
	GenesisStaticFileID        services.StaticFileID   `json:"genesis_static_file_id"`
	PasswordStaticFileID       services.StaticFileID   `json:"password_static_file_id"`
	SignerKeystoreStaticFileID services.StaticFileID   `json:"signer_keystore_static_file_id"`
}

func (test MyAdvancedTest) Configure(builder *testsuite.TestConfigurationBuilder) {
	builder.WithSetupTimeoutSeconds(
		240,
	).WithRunTimeoutSeconds(
		240,
	)
}

func (test MyAdvancedTest) Setup(networkCtx *networks.NetworkContext) (networks.Network, error) {

	_, err := networkCtx.LoadLambda(lambdaID,"kurtosistech/ethereum-kurtosis-lambda","{}")
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred loading the Ethereum Kurtosis Lambda in the Testuite")
	}

	logrus.Info("The Ethereum Kurtosis Lambda has been successfully added in the test")

	return networkCtx, nil
}

func (test MyAdvancedTest) Run(uncastedNetwork networks.Network) error {
	// Necessary because Go doesn't have generics
	castedNetwork := uncastedNetwork.(*networks.NetworkContext)

	ethLambdaCtx, err := castedNetwork.GetLambdaContext(lambdaID)

	respJsonStr, err := ethLambdaCtx.Execute("{}")
	if err != nil {
		return stacktrace.Propagate(err, "And error occurred executing the Ethereum Kurtosis Lambda")
	}
	ethResult := new(EthereumKurtosisLambdaResult)
	if err := json.Unmarshal([]byte(respJsonStr), ethResult); err != nil {
		return stacktrace.Propagate(err, "An error occurred deserializing the Lambda response")
	}

	serviceCtx, err := castedNetwork.GetServiceContext(ethResult.BootnodeServiceID)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred getting the Ethereum Go Client service context")
	}
	logrus.Infof("Got service context for Ethereum Go Client service '%v'", serviceCtx.GetServiceID())

	gethClient, err := getClient(serviceCtx.GetIPAddress())
	if err != nil {
		return stacktrace.Propagate(err, "Failed to get a gethClient from bootnode.")
	}
	defer gethClient.Close()

	key, err := getPrivateKey(serviceCtx, ethResult)
	if err != nil {
		return stacktrace.Propagate(err, "Failed to get private key")
	}

	transactor, err := bind.NewKeyedTransactorWithChainID(key.PrivateKey, big.NewInt(15))
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}
	transactor.GasPrice = big.NewInt(5)
	address, tx, helloWorld, err := bindings.DeployHelloWorld(transactor, gethClient)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred deploying the HelloWorld contract on the ETH Network")
	}
	fmt.Printf("Contract pending deploy: 0x%x\n", address)
	fmt.Printf("Transaction waiting to be mined: 0x%x\n\n", tx.Hash())

	if err := waitUntilTransactionMined(gethClient, tx.Hash()); err != nil {
		return stacktrace.Propagate(err, "An error occurred waiting for the HelloWorld contract to be mined")
	}
	logrus.Info("Deployed Hello World contract")

	name, err := helloWorld.Greet(&bind.CallOpts{Pending: true})
	if err != nil {
		log.Fatalf("Failed to retrieve pending name: %v", err)
	}
	fmt.Println("Pending name:", name)

	listAccountsCmd := []string{
		"/bin/sh",
		"-c",
		fmt.Sprintf("geth attach data/geth.ipc --exec eth.accounts"),
	}

	exitCode, logOutput, err := serviceCtx.ExecCommand(listAccountsCmd)
	if err != nil {
		return stacktrace.Propagate(err, "Executing command returned an error with logs: %+v", string(*logOutput))
	}
	if exitCode != execCommandSuccessExitCode {
		return stacktrace.NewError("Executing command returned an failing exit code with logs: %+v", string(*logOutput))
	}

	return nil
}

// ====================================================================================================
//                                       Private helper functions
// ====================================================================================================
func getClient(ipAddress string) (*ethclient.Client, error) {
	url := fmt.Sprintf("http://%v:%v", ipAddress, rpcPort)
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the Golang Ethereum client")
	}
	return client, nil
}

func getPrivateKey(serviceCtx *services.ServiceContext, ethResult *EthereumKurtosisLambdaResult) (*keystore.Key, error) {
	staticFileAbsFilepaths, err := serviceCtx.LoadStaticFiles(map[services.StaticFileID]bool{
		ethResult.SignerKeystoreStaticFileID: true,
		ethResult.PasswordStaticFileID: true,
	})
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred loading the static files corresponding to keys '%v' and '%v'", ethResult.SignerKeystoreStaticFileID, ethResult.PasswordStaticFileID)
	}
	signerKeystoreFilepath, found := staticFileAbsFilepaths[ethResult.SignerKeystoreStaticFileID]
	if !found {
		return nil, stacktrace.Propagate(err, "No filepath found for key '%v'; this is a bug in Kurtosis!", signerKeystoreFilepath)
	}

	signerKeystoreContent, err := ioutil.ReadFile(signerKeystoreFilepath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error happens reading file '%v'", signerKeystoreFilepath)
	}

	json, err := ioutil.ReadAll(strings.NewReader(string(signerKeystoreContent)))
	if err != nil {
		return nil, stacktrace.Propagate(err,"An error occurred when trying to read content for filepath '%v'", signerKeystoreFilepath)
	}

	passwordFilepath, found := staticFileAbsFilepaths[ethResult.PasswordStaticFileID]
	if !found {
		return nil, stacktrace.Propagate(err, "No filepath found for key '%v'; this is a bug in Kurtosis!", passwordFilepath)
	}

	passwordContent, err := ioutil.ReadFile(passwordFilepath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error happens reading file '%v'", passwordFilepath)
	}

	key, err := keystore.DecryptKey(json, string(passwordContent))
	if err != nil {
		return nil, stacktrace.Propagate(err,"An error occurred when trying to decrypt the private key")
	}
	return key, nil
}

func waitUntilTransactionMined(validatorClient *ethclient.Client, transactionHash common.Hash) error {
	for i := 0; i < maxNumCheckTransactionMinedRetries; i++ {
		receipt, err := validatorClient.TransactionReceipt(context.Background(), transactionHash)
		if err == nil && receipt != nil && receipt.BlockNumber != nil {
			return nil
		}
		if i < maxNumCheckTransactionMinedRetries-1 {
			time.Sleep(timeBetweenCheckTransactionMinedRetries)
		}
	}
	return stacktrace.NewError(
		"Transaction with hash '%v' wasn't mined even after checking %v times with %v between checks",
		transactionHash.Hex(),
		maxNumCheckTransactionMinedRetries,
		timeBetweenCheckTransactionMinedRetries)
}
