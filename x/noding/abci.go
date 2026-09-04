package noding

import (
	"context"
	"errors"

	errorsmod "cosmossdk.io/errors"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/arterynetwork/artr/x/noding/types"
)

// BeginBlocker начисляет награду предложившему блок, ведёт счёт
// подписей и наказывает за византийское поведение.
//
// В ABCI 2.0 обработчик не получает RequestBeginBlock: BeginBlock и
// EndBlock свёрнуты в FinalizeBlock, а всё нужное приходит через
// контекст. Предложивший блок берётся из заголовка, голоса предыдущего
// блока — из VoteInfos, свидетельства — из CometInfo.
func BeginBlocker(goCtx context.Context, k Keeper) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	proposer := ctx.BlockHeader().ProposerAddress
	if err := payProposerReward(ctx, proposer, k); err != nil {
		k.Logger(ctx).Error(
			"Couldn't pay proposer reward",
			"address", proposer,
			"error", err,
		)
	}
	votes := ctx.VoteInfos()
	if err := markStrokesAndTicks(ctx, votes, k); err != nil {
		k.Logger(ctx).Error(
			"Couldn't update statistics",
			"votes", votes,
			"error", err,
		)
	}
	evidence := misbehaviourOf(ctx)
	if err := punishWrongdoers(ctx, evidence, k); err != nil {
		k.Logger(ctx).Error(
			"Byzantine behavior detected",
			"evidences", evidence,
			"error", err,
		)
	}
	return nil
}

// misbehaviourOf достаёт свидетельства о нарушениях из контекста.
//
// CometInfo отдаёт их через интерфейс, а хранилище x/noding записывает
// abci.Misbehavior — поэтому здесь обратное преобразование, а не смена
// формата состояния.
func misbehaviourOf(ctx sdk.Context) []abci.Misbehavior {
	info := ctx.CometInfo()
	if info == nil {
		return nil
	}
	evidence := info.GetEvidence()
	out := make([]abci.Misbehavior, 0, evidence.Len())
	for i := 0; i < evidence.Len(); i++ {
		e := evidence.Get(i)
		v := e.Validator()
		out = append(out, abci.Misbehavior{
			Type:             abci.MisbehaviorType(e.Type()),
			Height:           e.Height(),
			Time:             e.Time(),
			TotalVotingPower: e.TotalVotingPower(),
			Validator: abci.Validator{
				Address: v.Address(),
				Power:   v.Power(),
			},
		})
	}
	return out
}

// EndBlocker пересобирает набор валидаторов.
func EndBlocker(goCtx context.Context, k Keeper) ([]abci.ValidatorUpdate, error) {
	return k.GatherValidatorUpdates(sdk.UnwrapSDKContext(goCtx))
}

func findValidatorAccAddress(ctx sdk.Context, k Keeper, validator abci.Validator) (sdk.AccAddress, error) {
	consAddr := sdk.ConsAddress(validator.Address)
	accAddr, found, _, err := k.GetValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		return accAddr, errorsmod.Wrap(err, "couldn't find validator")
	}
	if !found {
		return nil, errors.New("validator not found for consensus address " + consAddr.String())
	}
	return accAddr, nil
}

// punishWrongdoers - records infractions to the store and ban validators if needed
func punishWrongdoers(ctx sdk.Context, evz []abci.Misbehavior, k Keeper) error {
	for _, ev := range evz {
		accAddr, err := findValidatorAccAddress(ctx, k, ev.Validator)
		if err != nil {
			return err
		}
		err = k.MarkByzantine(ctx, accAddr, ev)
		if err != nil {
			return err
		}
	}
	return nil
}

// markStrokesAndTicks - increments signed/missed block counter and jail validators if needed
func markStrokesAndTicks(ctx sdk.Context, votes []abci.VoteInfo, k Keeper) error {
	for _, vote := range votes {
		accAddr, err := findValidatorAccAddress(ctx, k, vote.Validator)
		if err != nil {
			return err
		}
		// В CometBFT 0.38 булево SignedLastBlock заменил трёхзначный
		// флаг: подписал за блок, проголосовал за пустоту, отсутствовал.
		// Пропуском считается только отсутствие — так же решил и сам SDK
		// в x/slashing, и это ближе к прежнему смыслу: в Tendermint 0.34
		// подпись была и при нулевом голосе.
		if vote.BlockIdFlag != cmtproto.BlockIDFlagAbsent {
			err = k.MarkTick(ctx, accAddr)
		} else {
			err = k.MarkStroke(ctx, accAddr)
		}
		if err != nil {
			return errorsmod.Wrap(err, "cannot count a block for account "+accAddr.String())
		}
	}
	return nil
}

func payProposerReward(ctx sdk.Context, consAddr sdk.ConsAddress, k Keeper) error {
	accAddr, found, _, err := k.GetValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		return err
	}
	if !found {
		return types.ErrNotFound
	}
	if err = k.PayProposerReward(ctx, accAddr); err != nil {
		return err
	}
	return nil
}
