package schedule

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/arterynetwork/artr/x/schedule/keeper"
)

// BeginBlocker выполняет задачи, чей срок наступил.
//
// В SDK 0.50 обработчик принимает context.Context и возвращает ошибку:
// ABCI 2.0 свёл BeginBlock и EndBlock в FinalizeBlock, и данные о блоке
// приходят не аргументом, а через контекст.
func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	k.PerformSchedule(sdk.UnwrapSDKContext(ctx))
	return nil
}
