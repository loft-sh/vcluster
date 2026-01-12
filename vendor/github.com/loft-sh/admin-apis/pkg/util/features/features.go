package features

import (
	"fmt"
	"maps"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	stripe "github.com/stripe/stripe-go/v81"
	stripeclient "github.com/stripe/stripe-go/v81/client"
)

const (
	metadataQueryFmt = "metadata['%s']:'%s'"
)

type SyncedFeature struct {
	Feature *stripe.EntitlementsFeature
	Product *stripe.Product
}

func EnsureFeatures(
	stripeClient *stripeclient.API,
	syncedFeatures map[string]SyncedFeature,
	features []*licenseapi.Feature,
	additionalAttachAllProducts []stripe.Product,
	isLimit bool,
) error {
	err := EnsureStripeFeatures(stripeClient, syncedFeatures, features, isLimit)
	if err != nil {
		return err
	}

	if err = EnsureFeatureProducts(stripeClient, syncedFeatures); err != nil {
		return err
	}

	if !isLimit {
		if err = EnsureAttachAll(stripeClient, syncedFeatures, additionalAttachAllProducts); err != nil {
			return err
		}
	}
	return nil
}

func EnsureStripeFeatures(stripeClient *stripeclient.API, syncedFeatures map[string]SyncedFeature, features []*licenseapi.Feature, isLimit bool) error {
	for _, f := range features {
		err := EnsureFeatureExists(stripeClient, syncedFeatures, f.Name, f.DisplayName, BuildMetadata(f, isLimit, nil))
		if err != nil {
			return err
		}

		if f.Preview {
			err = EnsureFeatureExists(stripeClient, syncedFeatures, f.Name+"-preview", f.DisplayName+" [Preview]", BuildMetadata(f, isLimit, map[string]string{licenseapi.MetadataKeyFeatureIsPreview: licenseapi.MetadataValueTrue}))
			if err != nil {
				return err
			}
		}

		if isLimit {
			err = EnsureFeatureExists(stripeClient, syncedFeatures, f.Name+"-active", f.DisplayName+" [Active]", BuildMetadata(f, isLimit, map[string]string{licenseapi.MetadataKeyFeatureLimitTypeActive: licenseapi.MetadataValueTrue}))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func BuildMetadata(feature *licenseapi.Feature, isLimit bool, extraMetadata map[string]string) map[string]string {
	metadata := map[string]string{}
	if isLimit {
		metadata[licenseapi.MetadataKeyFeatureIsLimit] = licenseapi.MetadataValueTrue
		feature.Name = licenseapi.LimitsPrefix + feature.Name
	}

	if feature.Status == string(licenseapi.FeatureStatusHidden) {
		metadata[licenseapi.MetadataKeyFeatureIsHidden] = licenseapi.MetadataValueTrue
	}

	for key, value := range extraMetadata {
		metadata[key] = value
	}

	return metadata
}

func EnsureFeatureExists(stripeClient *stripeclient.API, syncedFeatures map[string]SyncedFeature, name, displayName string, extraMetadata map[string]string) error {
	feat, err := FindFeature(stripeClient, name)
	if err != nil {
		return err
	}

	if feat != nil {
		syncedFeatures[feat.Feature.ID] = *feat
		return nil
	}

	params := stripe.EntitlementsFeatureParams{
		Name:      &displayName,
		LookupKey: &name,
	}

	for key, value := range extraMetadata {
		params.AddMetadata(key, value)
	}

	feature, err := stripeClient.EntitlementsFeatures.New(&params)
	if err != nil {
		return fmt.Errorf("failed to create Stripe feature from feature %s: %v\n", *params.LookupKey, err)
	}
	syncedFeatures[feature.ID] = SyncedFeature{
		Feature: feature,
	}
	return nil
}

func FindFeature(stripeClient *stripeclient.API, id string) (*SyncedFeature, error) {
	list := stripeClient.EntitlementsFeatures.List(&stripe.EntitlementsFeatureListParams{
		LookupKey: &id,
	})
	if err := list.Err(); err != nil {
		return nil, fmt.Errorf("failed to list features while check if feature [%s] exists: %w", id, err)
	}
	if !list.Next() {
		return nil, nil
	}
	feature, ok := list.Current().(*stripe.EntitlementsFeature)
	if !ok {
		return nil, fmt.Errorf("failed to ")
	}
	feat := &SyncedFeature{
		Feature: feature,
	}
	return feat, nil
}

func EnsureFeatureProducts(stripeClient *stripeclient.API, syncedFeatures map[string]SyncedFeature) error {
	for key, feature := range syncedFeatures {
		if feature.Product != nil {
			continue
		}

		product, err := EnsureFeatureProduct(stripeClient, &feature)
		if err != nil {
			return err
		}
		feature.Product = product
		syncedFeatures[key] = feature
	}
	return nil
}

func EnsureFeatureProduct(stripeClient *stripeclient.API, syncedFeature *SyncedFeature) (*stripe.Product, error) {
	productSearch := stripeClient.Products.Search(&stripe.ProductSearchParams{
		SearchParams: stripe.SearchParams{
			Query: fmt.Sprintf(metadataQueryFmt, licenseapi.MetadataKeyProductForFeature, syncedFeature.Feature.LookupKey),
		},
	})
	if err := productSearch.Err(); err != nil {
		return nil, err
	}
	if productSearch.Next() {
		// a product exists with features name
		return productSearch.Product(), nil
	}

	usdCurrencyCode := "usd"
	unit := int64(2000000) // sample placeholder price
	interval := "year"
	intervalCount := int64(1)
	product, err := stripeClient.Products.New(&stripe.ProductParams{
		Name: &syncedFeature.Feature.Name,
		DefaultPriceData: &stripe.ProductDefaultPriceDataParams{
			Currency:   &usdCurrencyCode,
			UnitAmount: &unit,
			Recurring: &stripe.ProductDefaultPriceDataRecurringParams{
				Interval:      &interval,
				IntervalCount: &intervalCount,
			},
		},
		Metadata: map[string]string{
			licenseapi.MetadataKeyProductForFeature: syncedFeature.Feature.LookupKey,
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = stripeClient.ProductFeatures.New(&stripe.ProductFeatureParams{
		Product:            &product.ID,
		EntitlementFeature: &syncedFeature.Feature.ID,
	})
	if err != nil {
		return nil, err
	}
	return product, nil
}

func EnsureAttachAll(
	stripeClient *stripeclient.API,
	features map[string]SyncedFeature,
	attachTo []stripe.Product,
) error {
	if attachTo == nil {
		attachTo = []stripe.Product{}
	}
	productSearch := stripeClient.Products.Search(&stripe.ProductSearchParams{
		SearchParams: stripe.SearchParams{
			Query: fmt.Sprintf(metadataQueryFmt, licenseapi.MetadataKeyAttachAll, licenseapi.MetadataValueTrue),
		},
	})
	if err := productSearch.Err(); err != nil {
		return err
	}
	for productSearch.Next() {
		prod := productSearch.Product()
		attachTo = append(attachTo, *prod)
	}

	prodIDs := map[string]bool{}

	for _, prod := range attachTo {
		alreadyDone, ok := prodIDs[prod.ID]
		if ok && alreadyDone {
			continue
		}
		prodIDs[prod.ID] = true

		featuresToCheck := maps.Clone(features)

		if err := SearchProductForFeatures(stripeClient, prod.ID, featuresToCheck); err != nil {
			return err
		}
		for featureID, feat := range featuresToCheck {
			metadataHiddenValue, ok := feat.Feature.Metadata[licenseapi.MetadataKeyFeatureIsHidden]
			if !ok || metadataHiddenValue != licenseapi.MetadataValueTrue {
				_, err := stripeClient.ProductFeatures.New(&stripe.ProductFeatureParams{
					Product:            &prod.ID,
					EntitlementFeature: &featureID,
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func SearchProductForFeatures(stripeClient *stripeclient.API, productID string, featuresToCheck map[string]SyncedFeature) error {
	productFeaturesList := stripeClient.ProductFeatures.List(&stripe.ProductFeatureListParams{
		Product: &productID,
	})
	for productFeaturesList.Next() {
		if err := productFeaturesList.Err(); err != nil {
			return err
		}
		delete(featuresToCheck, productFeaturesList.ProductFeature().EntitlementFeature.ID)
	}
	return nil
}

func GetFeatureDisplayName(featureName string) (string, error) {
	features := licenseapi.GetAllFeatures()

	for _, feature := range features {
		if feature.Name == featureName {
			return feature.DisplayName, nil
		}
	}
	return "", fmt.Errorf("no feature found with name %s", featureName)
}
