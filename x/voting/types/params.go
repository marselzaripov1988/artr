package types

import (
	"fmt"
	"gopkg.in/yaml.v3"

	"github.com/pkg/errors"

	params "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Default parameter namespace
const (
	DefaultParamspace = ModuleName

	// DefaultVotingPeriod — срок голосования по умолчанию, в ЧАСАХ.
	//
	// Единица измерения — часы, а не блоки: Keeper.Propose считает конец
	// голосования как VotingPeriod * time.Hour, и validateVotingPeriod
	// говорит о том же ("at least 1 hour").
	//
	// Здесь стояло util.BlocksOneDay — число блоков (2880), подставленное
	// в поле часов. Замысел читается: 2880 блоков по 30 секунд и есть
	// сутки. Но в часах сутки — это 24, а 2880 часов дают 120 дней.
	//
	// Мейннет от этого не пострадал: у него значение лежит в состоянии и
	// равно 24. Умолчание срабатывает только там, где параметров в
	// генезисе нет вовсе — на новых сетях и стендах.
	DefaultVotingPeriod int32 = 24
)

// Parameter store keys
var (
	KeyParamVotingPeriod = []byte("VotingPeriod")
	KeyParamPollPeriod   = []byte("PollPeriod")
)

// ParamKeyTable for voting module
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params object
func NewParams(votingPeriod, pollPeriod int32) Params {
	return Params{
		VotingPeriod: votingPeriod,
		PollPeriod:   pollPeriod,
	}
}

// String implements the stringer interface for Params
func (p Params) String() string {
	bz, err := yaml.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(bz)
}

// ParamSetPairs - Implements params.ParamSet
func (p *Params) ParamSetPairs() params.ParamSetPairs {
	return params.ParamSetPairs{
		params.NewParamSetPair(KeyParamVotingPeriod, &p.VotingPeriod, validateVotingPeriod),
		params.NewParamSetPair(KeyParamPollPeriod, &p.PollPeriod, validateVotingPeriod),
	}
}

// DefaultParams defines the parameters for this module
func DefaultParams() Params {
	return NewParams(DefaultVotingPeriod, DefaultVotingPeriod)
}

func (p Params) Validate() error {
	if err := validateVotingPeriod(p.VotingPeriod); err != nil {
		return errors.Wrap(err, "invalid voting_period")
	}
	if err := validateVotingPeriod(p.PollPeriod); err != nil {
		return errors.Wrap(err, "invalid poll_period")
	}
	return nil
}

func validateVotingPeriod(i interface{}) error {
	v, ok := i.(int32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v < 1 {
		return fmt.Errorf("validating period must be at least 1 hour: %d", v)
	}

	return nil
}
