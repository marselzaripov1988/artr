package types

import (
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	errorsmod "cosmossdk.io/errors"
	params "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/arterynetwork/artr/util"
	"github.com/arterynetwork/artr/x/referral"
)

// Default parameter namespace
const (
	DefaultParamspace        = ModuleName
	DefaultMaxValidators     = 100
	DefaultJailAfter         = 2
	DefaultUnjailAfter       = util.BlocksOneHour
	DefaultLotteryValidators = 0
	DefaultMinStatus         = referral.StatusLeader
	DefaultMinSelfStake      = 10_000_000000
	DefaultMinTotalStake     = 50_000_000000
)

// DefaultMisbehaviourPenalty — доля собственной делегации, изымаемая за
// византийское поведение.
//
// Пять процентов взяты как в Cosmos за двойную подпись. Величину стоит
// пересмотреть: у Artery нарушитель и так получает пожизненный бан, а
// порог собственной делегации в мейннете 50 000 ARTR — пять процентов от
// него выходят скорее знаком, чем сдерживателем. Значение вынесено в
// параметры и меняется голосованием.
var DefaultMisbehaviourPenalty = util.Percent(5)

// Parameter store keys
var (
	DefaultMinCriteria = MinCriteria{
		Status:     DefaultMinStatus,
		SelfStake:  DefaultMinSelfStake,
		TotalStake: DefaultMinTotalStake,
	}

	DefaultVotingPower = Distribution{
		Slices: []Distribution_Slice{
			{
				Part:        util.Percent(15),
				VotingPower: 15,
			}, {
				Part:        util.Percent(85),
				VotingPower: 10,
			},
		},
		LuckiesVotingPower: 10,
	}

	KeyMaxValidators       = []byte("MaxValidators")
	KeyJailAfter           = []byte("JailAfter")
	KeyUnjailAfter         = []byte("UnjailAfter")
	KeyLotteryValidators   = []byte("LotteryValidators")
	KeyMinStatus           = []byte("MinStatus")
	KeyMinCriteria         = []byte("MinCriteria")
	KeyVotingPower         = []byte("VotingPower")
	KeyMisbehaviourPenalty = []byte("MisbehaviourPenalty")
)

// ParamKeyTable for noding module
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params object
func NewParams(
	maxValidators, jailAfter, unjailAfter, lotteryValidators uint32,
	minCriteria MinCriteria, misbehaviourPenalty util.Fraction,
) Params {
	return Params{
		MaxValidators:       maxValidators,
		JailAfter:           jailAfter,
		UnjailAfter:         unjailAfter,
		LotteryValidators:   lotteryValidators,
		MinCriteria:         minCriteria,
		MisbehaviourPenalty: misbehaviourPenalty,
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
		params.NewParamSetPair(KeyMaxValidators, &p.MaxValidators, validateMaxValidators),
		params.NewParamSetPair(KeyJailAfter, &p.JailAfter, validateJailAfter),
		params.NewParamSetPair(KeyUnjailAfter, &p.UnjailAfter, validateUnjailAfter),
		params.NewParamSetPair(KeyLotteryValidators, &p.LotteryValidators, validateAdditionalValidators),
		params.NewParamSetPair(KeyMinCriteria, &p.MinCriteria, validateMinCriteria),
		params.NewParamSetPair(KeyVotingPower, &p.VotingPower, validateVotingPower),
		params.NewParamSetPair(KeyMisbehaviourPenalty, &p.MisbehaviourPenalty, validateMisbehaviourPenalty),
	}
}

// DefaultParams defines the parameters for this module
func DefaultParams() Params {
	return NewParams(
		DefaultMaxValidators,
		DefaultJailAfter,
		DefaultUnjailAfter,
		DefaultLotteryValidators,
		DefaultMinCriteria,
		DefaultMisbehaviourPenalty,
	)
}

func validateMaxValidators(value interface{}) error {
	x, ok := value.(uint32)
	if !ok {
		return fmt.Errorf("invalid max_validators type: %T", value)
	}
	if x == 0 {
		return fmt.Errorf("max_validators must be positive: %d", x)
	}
	return nil
}

func validateAdditionalValidators(value interface{}) error {
	_, ok := value.(uint32)
	if !ok {
		return fmt.Errorf("invalid lottery_validators type: %T", value)
	}
	return nil
}

func validateJailAfter(value interface{}) error {
	x, ok := value.(uint32)
	if !ok {
		return fmt.Errorf("invalid jail_after type: %T", value)
	}
	if x == 0 {
		return fmt.Errorf("jail after must be positive: %d", x)
	}
	return nil
}

func validateUnjailAfter(value interface{}) error {
	x, ok := value.(uint32)
	if !ok {
		return fmt.Errorf("invalid unjail_after type: %T", value)
	}
	if x <= 0 {
		return fmt.Errorf("ujail after must be positive: %d", x)
	}
	return nil
}

func validateStatus(i interface{}) error {
	status, ok := i.(referral.Status)
	if !ok {
		return fmt.Errorf("invalid min_status type (uint8 expected): %T", i)
	}
	if err := status.Validate(); err != nil {
		return errors.Wrap(err, "invalid min_status parameter:")
	}
	return nil
}

func validateMinCriteria(i interface{}) error {
	mc, ok := i.(MinCriteria)
	if !ok {
		return fmt.Errorf("invalid min_criteria type: %T", i)
	}
	if err := mc.Validate(); err != nil {
		return errors.Wrap(err, "invalid min_criteria parameter:")
	}
	return nil
}

func validateVotingPower(i interface{}) error {
	distr, ok := i.(Distribution)
	if !ok {
		return errors.Errorf("invalid voting_power type: %T", i)
	}
	return errors.Wrap(distr.Validate(), "invalid voting_power")
}

func (p *Params) Validate() error {
	if p == nil {
		return fmt.Errorf("params are nil")
	}
	if err := validateMaxValidators(p.MaxValidators); err != nil {
		return errorsmod.Wrap(err, "invalid MaxValidators")
	}
	if err := validateJailAfter(p.JailAfter); err != nil {
		return errorsmod.Wrap(err, "invalid JailAfter")
	}
	if err := validateUnjailAfter(p.UnjailAfter); err != nil {
		return errorsmod.Wrap(err, "invalid UnjailAfter")
	}
	if err := validateAdditionalValidators(p.LotteryValidators); err != nil {
		return errorsmod.Wrap(err, "invalid LotteryValidators")
	}
	if err := validateMinCriteria(p.MinCriteria); err != nil {
		return errorsmod.Wrap(err, "invalid MinCriteria")
	}
	if err := validateVotingPower(p.VotingPower); err != nil {
		return err
	}
	if err := validateMisbehaviourPenalty(p.MisbehaviourPenalty); err != nil {
		return errorsmod.Wrap(err, "invalid MisbehaviourPenalty")
	}
	return nil
}

// validateMisbehaviourPenalty — доля в границах от нуля до единицы
// включительно.
//
// Ноль допустим: это нынешнее поведение сети, где за византийское
// поведение не берут ничего. Единица тоже: изъять всю собственную
// делегацию — крайняя, но осмысленная мера.
func validateMisbehaviourPenalty(value interface{}) error {
	x, ok := value.(util.Fraction)
	if !ok {
		return fmt.Errorf("invalid misbehaviour_penalty type: %T", value)
	}
	if x.IsNullValue() {
		return fmt.Errorf("misbehaviour_penalty is not set")
	}
	if x.IsNegative() || x.GT(util.FractionInt(1)) {
		return fmt.Errorf("misbehaviour_penalty must be in [0; 1]: %s", x.String())
	}
	return nil
}
