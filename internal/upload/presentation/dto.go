package presentation

type uploadResponse struct {
	URL          string            `json:"url"`
	CropVariants map[string]string `json:"cropVariants,omitempty"`
}
