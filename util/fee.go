package util

import (
	"cosmossdk.io/math"
)

func CalculateFee(amount math.Int, txFeeFraction Fraction, txFeeMaxAmount int64, forProposerFeeFraction, forCompanyFeeFraction Fraction) math.Int {
	fee := math.NewInt(txFeeFraction.MulInt64(amount.Int64()).Int64())

	maxFee := math.NewInt(txFeeMaxAmount)
	if !maxFee.IsZero() && fee.GT(maxFee) {
		fee = maxFee
	}

	return calculateSplittableFee(fee, forProposerFeeFraction, forCompanyFeeFraction)
}

func calculateForBurningFeeFraction(forProposerFeeFraction, forCompanyFeeFraction Fraction) Fraction {
	return FractionInt(1).Sub(forProposerFeeFraction).Sub(forCompanyFeeFraction)
}

func CalculateTransactionFeeSplitRatiosLCM(forProposerFeeFraction, forCompanyFeeFraction Fraction) math.Int {
	return math.NewIntFromBigInt(lcm(lcm(forProposerFeeFraction.denom, forCompanyFeeFraction.denom), calculateForBurningFeeFraction(forProposerFeeFraction, forCompanyFeeFraction).denom))
}

func calculateSplittableFee(feeLimit math.Int, forProposerFeeFraction, forCompanyFeeFraction Fraction) math.Int {
	return feeLimit.Sub(feeLimit.Mod(CalculateTransactionFeeSplitRatiosLCM(forProposerFeeFraction, forCompanyFeeFraction)))
}

func SplitFee(splittableFee math.Int, forProposerFeeFraction, forCompanyFeeFraction Fraction) (forProposer, forCompany, forBurning math.Int) {
	return math.NewInt(forProposerFeeFraction.MulInt64(splittableFee.Int64()).Int64()),
		math.NewInt(forCompanyFeeFraction.MulInt64(splittableFee.Int64()).Int64()),
		math.NewInt(calculateForBurningFeeFraction(forProposerFeeFraction, forCompanyFeeFraction).MulInt64(splittableFee.Int64()).Int64())
}

func IsSendable(denom string) bool {
	return denom == ConfigMainDenom
}
