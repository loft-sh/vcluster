//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Simulated data for the card_present payment method.
type TestHelpersTerminalReaderPresentPaymentMethodCardPresentParams struct {
	// The card number, as a string without any separators.
	Number *string `form:"number"`
}

// Simulated data for the interac_present payment method.
type TestHelpersTerminalReaderPresentPaymentMethodInteracPresentParams struct {
	// Card Number
	Number *string `form:"number"`
}

// Presents a payment method on a simulated reader. Can be used to simulate accepting a payment, saving a card or refunding a transaction.
type TestHelpersTerminalReaderPresentPaymentMethodParams struct {
	Params `form:"*"`
	// Simulated on-reader tip amount.
	AmountTip *int64 `form:"amount_tip"`
	// Simulated data for the card_present payment method.
	CardPresent *TestHelpersTerminalReaderPresentPaymentMethodCardPresentParams `form:"card_present"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Simulated data for the interac_present payment method.
	InteracPresent *TestHelpersTerminalReaderPresentPaymentMethodInteracPresentParams `form:"interac_present"`
	// Simulated payment type.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTerminalReaderPresentPaymentMethodParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}
