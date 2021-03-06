/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

                 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provisional

import (
	"fmt"

	"github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric/common/configtx"
	configtxchannel "github.com/hyperledger/fabric/common/configtx/handlers/channel"
	configtxorderer "github.com/hyperledger/fabric/common/configtx/handlers/orderer"
	"github.com/hyperledger/fabric/common/genesis"
	"github.com/hyperledger/fabric/orderer/common/bootstrap"
	"github.com/hyperledger/fabric/orderer/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
)

// Generator can either create an orderer genesis block or config template
type Generator interface {
	bootstrap.Helper

	// ChannelTemplate returns a template which can be used to help initialize a channel
	ChannelTemplate() configtx.Template
}

const (
	// ConsensusTypeSolo identifies the solo consensus implementation.
	ConsensusTypeSolo = "solo"
	// ConsensusTypeKafka identifies the Kafka-based consensus implementation.
	ConsensusTypeKafka = "kafka"
	// ConsensusTypeSbft identifies the SBFT consensus implementation.
	ConsensusTypeSbft = "sbft"

	// TestChainID is the default value of ChainID. It is used by all testing
	// networks. It it necessary to set and export this variable so that test
	// clients can connect without being rejected for targetting a chain which
	// does not exist.
	TestChainID = "testchainid"

	// AcceptAllPolicyKey is the key of the AcceptAllPolicy.
	AcceptAllPolicyKey = "AcceptAllPolicy"
)

// DefaultChainCreationPolicyNames is the default value of ChainCreatorsKey.
var DefaultChainCreationPolicyNames = []string{AcceptAllPolicyKey}

type bootstrapper struct {
	minimalItems     []*cb.ConfigItem
	minimalGroups    []*cb.ConfigGroup
	systemChainItems []*cb.ConfigItem
}

// New returns a new provisional bootstrap helper.
func New(conf *config.TopLevel) Generator {
	bs := &bootstrapper{
		minimalItems: []*cb.ConfigItem{
			// Orderer Config Types
			configtxorderer.TemplateConsensusType(conf.Genesis.OrdererType),
			configtxorderer.TemplateBatchSize(&ab.BatchSize{
				MaxMessageCount:   conf.Genesis.BatchSize.MaxMessageCount,
				AbsoluteMaxBytes:  conf.Genesis.BatchSize.AbsoluteMaxBytes,
				PreferredMaxBytes: conf.Genesis.BatchSize.PreferredMaxBytes,
			}),
			configtxorderer.TemplateBatchTimeout(conf.Genesis.BatchTimeout.String()),
			configtxorderer.TemplateIngressPolicyNames([]string{AcceptAllPolicyKey}),
			configtxorderer.TemplateEgressPolicyNames([]string{AcceptAllPolicyKey}),
		},

		minimalGroups: []*cb.ConfigGroup{
			// Chain Config Types
			configtxchannel.DefaultHashingAlgorithm(),
			configtxchannel.DefaultBlockDataHashingStructure(),
			configtxchannel.TemplateOrdererAddresses([]string{fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort)}),

			// Policies
			cauthdsl.TemplatePolicy(configtx.NewConfigItemPolicyKey, cauthdsl.RejectAllPolicy),
			cauthdsl.TemplatePolicy(AcceptAllPolicyKey, cauthdsl.AcceptAllPolicy),
		},

		systemChainItems: []*cb.ConfigItem{
			configtxorderer.TemplateChainCreationPolicyNames(DefaultChainCreationPolicyNames),
		},
	}

	switch conf.Genesis.OrdererType {
	case ConsensusTypeSolo, ConsensusTypeSbft:
	case ConsensusTypeKafka:
		bs.minimalItems = append(bs.minimalItems, configtxorderer.TemplateKafkaBrokers(conf.Kafka.Brokers))
	default:
		panic(fmt.Errorf("Wrong consenter type value given: %s", conf.Genesis.OrdererType))
	}

	return bs
}

func (bs *bootstrapper) ChannelTemplate() configtx.Template {
	return configtx.NewCompositeTemplate(
		configtx.NewSimpleTemplate(bs.minimalItems...),
		configtx.NewSimpleTemplateNext(bs.minimalGroups...),
	)
}

func (bs *bootstrapper) GenesisBlock() *cb.Block {
	block, err := genesis.NewFactoryImpl(
		configtx.NewCompositeTemplate(
			configtx.NewSimpleTemplate(bs.systemChainItems...),
			bs.ChannelTemplate(),
		),
	).Block(TestChainID)

	if err != nil {
		panic(err)
	}
	return block
}
