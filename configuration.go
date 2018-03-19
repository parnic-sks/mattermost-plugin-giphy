package main

type GiphyPluginConfiguration struct {
	Rating             string
	Language           string
	Rendition          string
	APIKey             string
	SingleGIFTrigger   string
	MultipleGIFTrigger string
}

var defaultConfig = GiphyPluginConfiguration{
	SingleGIFTrigger:   "giphy",
	MultipleGIFTrigger: "gifs",
	Rating:             "",
	Language:           "",
	Rendition:          "fixed_height_small",
	APIKey:             "dc6zaTOxFJmzC",
}

func (c *GiphyPluginConfiguration) EnsureValidity() error {
	// Set mandatory fields that are empty  to default value
	if len(c.APIKey) == 0 {
		c.APIKey = defaultConfig.APIKey
	}
	if len(c.SingleGIFTrigger) == 0 {
		c.SingleGIFTrigger = defaultConfig.SingleGIFTrigger
	}
	if len(c.MultipleGIFTrigger) == 0 {
		c.MultipleGIFTrigger = defaultConfig.MultipleGIFTrigger
	}
	if len(c.Rendition) == 0 {
		c.Rendition = defaultConfig.Rendition
	}

	return nil
}
