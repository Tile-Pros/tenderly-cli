package truffle

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Contract struct {
	Name              string                     `json:"contractName"`
	Abi               interface{}                `json:"abi"`
	Bytecode          string                     `json:"bytecode"`
	DeployedBytecode  string                     `json:"deployedBytecode"`
	SourceMap         string                     `json:"sourceMap"`
	DeployedSourceMap string                     `json:"deployedSourceMap"`
	Source            string                     `json:"source"`
	SourcePath        string                     `json:"sourcePath"`
	Ast               ContractAst                `json:"legacyAST"`
	Compiler          ContractCompiler           `json:"compiler"`
	Networks          map[string]ContractNetwork `json:"networks"`

	SchemaVersion string    `json:"schemaVersion"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type ContractCompiler struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ContractNetwork struct {
	Events          interface{} `json:"events"`
	Links           interface{} `json:"links"`
	Address         string      `json:"address"`
	TransactionHash string      `json:"transactionHash"`
}

type Node struct {
	NodeType     string `json:"nodeType"`
	AbsolutePath string `json:"absolutePath"`
}

type ContractAst struct {
	AbsolutePath    string           `json:"absolutePath"`
	ExportedSymbols map[string][]int `json:"exportedSymbols"`
	Id              int              `json:"id"`
	NodeType        string           `json:"nodeType"`
	Nodes           []Node           `json:"nodes"`
	Src             string           `json:"src"`
}

type ApiContract struct {
	ID string `json:"id"`

	AccountID string `json:"account_id"`
	ProjectID string `json:"project_id"`

	NetworkID string `json:"network_id"`
	Public    bool   `json:"public"`

	Address string `json:"address"`

	Name string `json:"contract_name"`

	Abi       string `json:"abi"`
	Bytecode  string `json:"bytecode"`
	Source    string `json:"source"`
	SourceMap string `json:"source_map"`

	CreatedAt time.Time `json:"created_at"`
}

type ApiDeploymentInformation struct {
	NetworkID string `json:"network_id"`
	Address   string `json:"address"`
}

func GetTruffleContracts(buildDir string, networkIDs ...string) ([]Contract, int, error) {
	files, err := ioutil.ReadDir(buildDir)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed listing truffle build files")
	}

	networkIDFilterMap := make(map[string]bool)
	for _, networkID := range networkIDs {
		networkIDFilterMap[networkID] = true
	}

	hasNetworkFilters := len(networkIDFilterMap) > 0

	sources := make(map[string]bool)
	var contracts []Contract
	var numberOfContractsWithANetwork int
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(buildDir, file.Name())
		data, err := ioutil.ReadFile(filePath)

		if err != nil {
			return nil, 0, errors.Wrap(err, "failed reading truffle build file")
		}

		var contract Contract
		err = json.Unmarshal(data, &contract)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed parsing truffle build file")
		}

		sources[contract.SourcePath] = true
		for _, node := range contract.Ast.Nodes {
			if node.NodeType == "ImportDirective" && !sources[node.AbsolutePath] {
				sources[node.AbsolutePath] = false
			}
		}

		if len(contract.Networks) > 0 && hasNetworkFilters {
			for networkID := range contract.Networks {
				if !networkIDFilterMap[networkID] {
					delete(contract.Networks, networkID)
				}
			}

			if len(contract.Networks) == 0 {
				continue
			}
		}

		contracts = append(contracts, contract)
		numberOfContractsWithANetwork += len(contract.Networks)
	}

	for path, included := range sources {
		if !included {
			source, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, 0, errors.Wrap(err, "failed reading contract source file")
			}

			contracts = append(contracts, Contract{
				Source:     string(source),
				SourcePath: path,
			})
		}
	}

	return contracts, numberOfContractsWithANetwork, nil
}
