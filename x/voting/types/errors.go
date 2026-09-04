package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	// Signer not in government list
	ErrSignerNotAllowed          = errorsmod.Register(ModuleName, 1, "signer not in government list")
	ErrOtherActive               = errorsmod.Register(ModuleName, 2, "other proposal is active")
	ErrAlreadyVoted              = errorsmod.Register(ModuleName, 3, "already voted")
	ErrNoActiveProposals         = errorsmod.Register(ModuleName, 4, "no active proposals to vote")
	ErrProposalGovernorExists    = errorsmod.Register(ModuleName, 5, "candidate already in government list")
	ErrProposalGovernorNotExists = errorsmod.Register(ModuleName, 6, "candidate not in government list")
	ErrProposalGovernorLast      = errorsmod.Register(ModuleName, 7, "cannot remove the last governor")
	ErrNoActivePoll              = errorsmod.Register(ModuleName, 8, "no active poll")
	ErrRespondentNotAllowed      = errorsmod.Register(ModuleName, 9, "poll requirements don't match")
)
