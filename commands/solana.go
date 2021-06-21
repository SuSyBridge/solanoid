package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/portto/solana-go-sdk/common"
)

func ValidateError(t *testing.T, err error) {
	if err != nil {
		t.Logf("Error: %v \n", err)
		t.FailNow()
	}
}

func ValidateErrorExistence(t *testing.T, err error) {
	if err == nil {
		t.Logf("No error occured!")
		t.FailNow()
		return
	}

	t.Logf("Error: %v \n", err)
}


func SystemFaucet(t *testing.T, recipient string, amount uint64) error {
	t.Logf("transfer %v SOL to %v address \n", amount, recipient)

	cmd := exec.Command("solana", "transfer", recipient, fmt.Sprint(amount), "--allow-unfunded-recipient")

	output, err := cmd.CombinedOutput()
	t.Log(string(output))

	if err != nil {
		t.Log(err.Error())
		// log.Fatal(err)
		return err
	}

	// t.Log(output)
	
	return nil
}

func InferSystemDefinedRPC() (string, error) {
	cmd := exec.Command("solana", "config", "get")
	output, err := cmd.CombinedOutput()
	
	rgx, _ := regexp.Compile("RPC URL: .+")
	result := rgx.Find(output)
	resultStr := strings.Trim(string(result), "\n\r ")
	resultList := strings.Split(resultStr, " ")
	rpcURL := resultList[len(resultList) - 1]
	
	fmt.Println(resultList)
	rpcURL = strings.Trim(rpcURL, "\n\r")

	if err != nil {
		return "", err
	}

	// t.Log(output)
	return rpcURL, nil
}

type TokenCreateResult struct {
	Token     common.PublicKey
	Owner     common.PublicKey
	Signature string
}


func trimAndTakeLast(str, del string) string {
	resultStr := strings.Trim(str, "\n\r ")
	resultList := strings.Split(resultStr, del)
	lastEl := resultList[len(resultList) - 1]
	return lastEl
}
	
func CreateToken(ownerPrivateKeysPath string) (*TokenCreateResult, error) {
	decimals := 8
	cmd := exec.Command("spl-token", "create-token", "--owner", ownerPrivateKeysPath,  "--decimals", fmt.Sprintf("%v", decimals))
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, err
	}

	tokenCatchRegex, _ := regexp.Compile("Creating token .+")
	signatureCatchRegex, _ := regexp.Compile("Signature: .+")
	tokenAddress := trimAndTakeLast(string(tokenCatchRegex.Find(output)), " ")
	signature := trimAndTakeLast(string(signatureCatchRegex.Find(output)), " ")
	
	fmt.Println(tokenAddress)
	fmt.Println(signature)

	owner, err := ReadAccountAddress(ownerPrivateKeysPath)
	if err != nil {
		return nil, err
	}

	return &TokenCreateResult{
		Token: common.PublicKeyFromString(tokenAddress),
		Owner: common.PublicKeyFromString(owner),
		Signature: signature,
	}, nil
	// spl-token create-token --owner private-keys/main-deployer.json 
}

func ReadAccountAddress(privateKeysPath string) (string, error) {
	cmd := exec.Command("solana-keygen", "pubkey", privateKeysPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", err
	}
	result := string(output)
	account := strings.Trim(result, "\n\r ")
	
	fmt.Println(account)
	return account, nil
}

func ReadAccountBalance(address string) (float64, error) {
	cmd := exec.Command("solana", "balance", address)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return 0, err
	}
	result := string(output)
	resultStr := strings.Trim(string(result), "\n\r ")
	resultList := strings.Split(resultStr, " ")
	balance := resultList[0]
	
	balance = strings.Trim(balance, "\n\r")
	castedBalance, err := strconv.ParseFloat(balance, 64)
	
	if err != nil {
		return 0, err
	}

	return castedBalance, nil
}

// AccountAddress, PDA, error
func CreatePersistentAccountWithPDA(path string, forceRewrite bool, seeds [][]byte) (common.PublicKey, common.PublicKey, error) {
	var err error
	err = CreatePersistedAccount(path, forceRewrite)
	if err != nil {
		return *new(common.PublicKey), *new(common.PublicKey), err
	}

	accountAddress, err := ReadAccountAddress(path)
	if err != nil {
		return *new(common.PublicKey), *new(common.PublicKey), err
	}

	var targetAddressPDA common.PublicKey
	targetAddressPDA, err = common.CreateProgramAddress(seeds, common.PublicKeyFromString(accountAddress))
	if err != nil {
		return CreatePersistentAccountWithPDA(path, forceRewrite, seeds)
	}
	return common.PublicKeyFromString(accountAddress), targetAddressPDA, nil
}

func CreatePersistedAccount(path string, forceRewrite bool) error {
	var forceArg string
	if forceRewrite {
		forceArg = "--force"
	}

	cmd := exec.Command("solana-keygen", "new", "-o", path, "--no-bip39-passphrase", forceArg)

	_, err := cmd.CombinedOutput()
	// t.Log(string(output))

	if err != nil {
		return err
	}

	return nil
}


func ReadSPLTokenBalance(ownerPrivateKeysPath, tokenProgramAddress string) (float64, error) {
	cmd := exec.Command("spl-token", "balance", "--owner", ownerPrivateKeysPath, tokenProgramAddress)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return 0, err
	}
	result := string(output)
	balance := strings.Trim(result, "\n\r")
	castedBalance, err := strconv.ParseFloat(balance, 64)
	
	if err != nil {
		return 0, err
	}

	return castedBalance, nil
}

// spl-token transfer <TOKEN_ADDRESS> <TOKEN_AMOUNT> <RECIPIENT_ADDRESS or RECIPIENT_TOKEN_ACCOUNT_ADDRESS> --config <PATH>
func TransferSPLTokens(tokenHolderPath, tokenAddress, recipientTokenAccountAddress, delegate string, amount float64) error {
	cmd := exec.Command("spl-token", "transfer", "--owner", tokenHolderPath, "--from", delegate, tokenAddress, fmt.Sprintf("%v", amount), recipientTokenAccountAddress)
	output, err := cmd.CombinedOutput()
	fmt.Printf(string(output))

	if err != nil {
		return err
	}

	return nil
}

// spl-token approve [FLAGS] [OPTIONS] <TOKEN_ACCOUNT_ADDRESS> <TOKEN_AMOUNT> <DELEGATE_TOKEN_ACCOUNT_ADDRESS>
func DelegateSPLTokenAmount(tokenOwnerPath, tokenAccountAddress, delegateTokenAccountAddress string, amount float64) error {
	cmd := exec.Command("spl-token", "approve", "--owner", tokenOwnerPath, tokenAccountAddress, fmt.Sprintf("%v", amount), delegateTokenAccountAddress)
	output, err := cmd.CombinedOutput()
	fmt.Printf(string(output))

	if err != nil {
		return err
	}

	return nil
}

// On mint we provide token program address & account data address
// spl-token mint --owner private-keys/token-owner.json $TOKEN_PROGRAM 10 GMuGCTYcCV7FiKg3kQ7LArfZQdhagvUYWNXb1DNZQSGK
func MintToken(minterPrivateKeysPath, tokenProgramAddress string, amount float64, tokenDataAccount string) error {
	cmd := exec.Command("spl-token", "mint", "--owner", minterPrivateKeysPath, tokenProgramAddress, fmt.Sprintf("%v", amount), tokenDataAccount)
	output, err := cmd.CombinedOutput()
	fmt.Printf(string(output))

	if err != nil {
		return err
	}

	return nil
}

// On burn - only token data account address
// spl-token burn GMuGCTYcCV7FiKg3kQ7LArfZQdhagvUYWNXb1DNZQSGK 1 --owner private-keys/token-owner.json 
func BurnToken(burnerPrivateKeysPath, tokenDataAccount string, amount float64) error {
	cmd := exec.Command("spl-token", "burn", "--owner", burnerPrivateKeysPath, tokenDataAccount, fmt.Sprintf("%v", amount))
	output, err := cmd.CombinedOutput()
	fmt.Printf(string(output))

	if err != nil {
		return err
	}

	return nil
}


func CreateTokenAccount(currentOwnerPrivateKeyPath, tokenAddress string) (string, error) {
	cmd := exec.Command("spl-token", "create-account", "--owner", currentOwnerPrivateKeyPath, tokenAddress)
	output, err := cmd.CombinedOutput()
	// t.Log(string(output))

	// Creating account GMuGCTYcCV7FiKg3kQ7LArfZQdhagvUYWNXb1DNZQSGK
	dataAccountCatchRegex, _ := regexp.Compile("Creating account .+")
	tokenDataAccount := trimAndTakeLast(string(dataAccountCatchRegex.Find(output)), " ")

	fmt.Println(tokenDataAccount)

	if err != nil {
		return "", err
	}

	return tokenDataAccount, nil
}

func AuthorizeToken(t *testing.T, currentOwnerPrivateKeyPath, tokenAddress, authority, recipient string) error {
	cmd := exec.Command("spl-token", "authorize", "--owner", currentOwnerPrivateKeyPath, tokenAddress, authority, recipient)
	output, err := cmd.CombinedOutput()
	t.Log(string(output))

	if err != nil {
		return err
	}

	return nil
}

func DeploySolanaProgram(t *testing.T, tag string, programPrivateKeysPath, deployerPrivateKeysPath, programBinaryPath string) (string, error) {
	t.Log("deploying program")

	cmd := exec.Command("solana", "program", "deploy", "--keypair", deployerPrivateKeysPath, "--program-id", programPrivateKeysPath, programBinaryPath)

	output, err := cmd.CombinedOutput()
	
	t.Log(string(output))

	outputList := strings.Split(string(output), " ")
	programID := outputList[len(outputList) - 1]
	programID = strings.Trim(programID, "\n\r")

	t.Logf("Program: %v; Deployed Program ID is: %v\n", tag, programID)
	// t.Logf("Program: %v; Deployed Program ID is: %v\n", tag, common.PublicKeyFromString(programID))

	if err != nil {
		return "", err
		// log.Fatal(err)
	}

	// t.Log(output)
	
	return programID, nil
}

func ReadPKFromPath(t *testing.T, path string) (string, error) {
	result, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	var input []byte

	err = json.Unmarshal(result, &input)
	if err != nil {
		return "", err
	}

	encodedPrivKey := base58.Encode(input)
	// t.Logf("priv key: %v \n", encodedPrivKey)

	return encodedPrivKey, nil
}