# V32 Migration Guide

Version 32 of stripe-go contains some very sizable breaking changes.

The major reason that we moved forward on them is that it was previously
impossible when encoding a parameter struct for an API call to make a
distinction between a field that a user had left unset versus a field that had
been set explicitly, but to an empty value. So if we had a parameter struct
like this:

``` go
type UsageRecordParams struct {
	Quantity uint64 `form:"quantity"`
}
```

We were unable to differentiate these two cases:

``` go
// Initialized with no quantity
UsageRecord {}

// Initialized with an explicitly zero
UsageRecord {
    Quantity: 0,
}
```

This is because any uninitialized fields on a struct in Go are set to their
type's "zero value", which for an integer is `0`.

Working around the problem required a secondary field to help explicitly state
that the zero value was intended, which was quite unintuitive for users:

``` go
type UsageRecordParams struct {
	Quantity     uint64 `form:"quantity"`
	QuantityZero bool   `form:"quantity,zero"`
}

UsageRecord {
    QuantityZero: true,
}
```

To address the problem, we moved every parameter struct over to use pointers
instead. So the above becomes:

``` go
type UsageRecordParams struct {
	Quantity *int64 `form:"quantity"`
}
```

Because in Go you can't take the address of an inline value (`&0`), we provide
a set of helper functions like `stripe.Int64` specifically for initializing
these structs:

``` go
UsageRecord {
    Quantity: stripe.Int64(0),
}
```

The zero value for pointers is `nil`, so we can now easily determine which
values on a struct were never set, and which ones were explicitly set to a zero
value, thus eliminating the need for the secondary fields like `QuantityZero`.

Because this is a large change, we also took the opportunity to do some
housekeeping throughout the library. Most of this involves renaming fields and
some resources to be more accurate according to how they're named in Stripe's
REST API, but it also involves some smaller changes like moving some types and
constants around.

Please see the list below for the complete set of changes.

## Major changes

* All fields on parameter structs (those that end with `*Params`) are now
  pointers. Please use the new helper functions to set them:
    * `stripe.Bool`
    * `stripe.Float64`
    * `stripe.Int64`
    * `stripe.String`

    This also means that extra fields that used to be solely used for tracking
    meaningful zero values like `CouponEmpty` and `QuantityZero` have been
    dropped. Use their corresponding field (e.g., `Coupon`, `Quantity`) with an
    explicit empty value instead (`stripe.String("")`, `stripe.Int64(0)`).
* Many fields have been renamed so that they're more consistent with their name
  in Stripe's REST API. Most of the time, this changes abbreviations to a more
  fully expanded form. For example:
    * `Desc` becomes `Description`.
    * `Live` becomes `Livemode`.
* A few names of API resources (and their corresponding parameter and list
  classes) have changed:
    * `Fee` becomes `ApplicationFee`.
    * `FeeRefund` becomes `ApplicationFeeRefund`.
    * `Owner` becomes `AdditionalOwner`.
    * `Sub` becomes `Subscription`.
    * `SubItem` becomes `SubscriptionItem`.
    * `Transaction` becomes `BalanceTransaction`.
    * `TxFee` becomes `BalanceTransactionFee`.
* Some sets of constants have been renamed and migrated to the top-level
  `stripe` package. All constants now have a prefix according to what they
  describe (for example, card brands all start with `CardBrand*` like
  `CardBrandVisa`) and all now reside in the `stripe` package (for example
  `dispute.Duplicate` is now `stripe.DisputeReasonDuplicate`).
* Some structs that used to be shared between requests and responses are now
  broken apart. All API calls should be using only structs that end with a
  `*Params` suffix. So for example, if you were using `Address` or `DOB`
  before, you should now use `AddressParams` and `DOBParams`.

## Other changes

* All integer values now use `int64` as their type. This means that the
  `stripe.Int64` helper function is appropriate for setting all integer values.
  This usually doesn't require a change because just setting these fields to a
  numerical literal didn't require that the type be explicitly stated.
* `Event.GetObjValue` becomes `Event.GetObjectValue`
* `Params.AddMeta` becomes `Params.AddMetadata`
* `Params.End` and `ListParams.End` become `EndingBefore`, and become a pointer
  (use `stripe.String` to set them as with other parameters).
* `Params.Expand` and `ListParams.Expand` (the fields) becomes a slice of
  pointers (instead of a slice of strings).
* `Params.Expand` and `ListParams.Expand` (the functions) become `AddExpand`.
* `Params.IdempotencyKey` becomes a pointer.
* `Params.Limit` becomes a pointer.
* `Params.Meta` becomes `Params.Metadata`
* `Params.Start` and `ListParams.Start` become `StartingAfter`
* `Params.StripeAccount` and `ListParams.StripeAccount` become pointers.
* List object data is now accessed with `object.Data` instead of `object.List`.
  Nothing changes if you were using iterators and `Next`.
* The previously deprecated `FileUploadParams.File` has been removed. Please
  use `FileUploadParams.FileReader` instead.
* The previously deprecated `Params.Account` has been removed. Please use
  `Params.StripeAccount` instead.

As usual, if you find bugs, please [open them on the repository][issues], or
reach out to `support@stripe.com` if you have any other questions.

[issues]: https://github.com/stripe/stripe-go/issues/new

<!--
# vim: set tw=79:
-->
