package urlreplacer

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUrlReplacerFactory(t *testing.T) {
	t.Run("shoudl return error when", func(t *testing.T) {
		tests := []struct {
			name         string
			mapping      map[string]string
			errorMesasge string
		}{
			{
				name:         "mappings is empty",
				mapping:      make(map[string]string),
				errorMesasge: "you must specify at least one mapping",
			},
			{
				name: "source url is incorrect",
				mapping: map[string]string{
					string(rune(0x7f)): "https://github.com",
				},
				errorMesasge: "falied to parse source url: parse \"\\u007f\": net/url: invalid control character in URL",
			},
			{
				name: "target url is incorrect ",
				mapping: map[string]string{
					"locahost": string(rune(0x7f)),
				},
				errorMesasge: "falied to parse target url: parse \"\\u007f\": net/url: invalid control character in URL",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual, err := NewUrlReplacerFactory(tt.mapping)

				assert.Nil(t, actual)
				assert.EqualError(t, err, tt.errorMesasge)
			})
		}
	})

	t.Run("shodul return replacer", func(t *testing.T) {
		actual, err := NewUrlReplacerFactory(map[string]string{
			"localhost": "https://github.com",
		})

		assert.NotNil(t, actual)
		assert.NoError(t, err)
	})
}

func TestUrlReplacerFactoryMake(t *testing.T) {
	factory, _ := NewUrlReplacerFactory(map[string]string{
		"http://localhost": "https://github.com",
	})

	t.Run("shoduld return error when mapping not found", func(t *testing.T) {
		parsedUrl, _ := url.Parse("http://unknow.com")

		actual, err := factory.Make(parsedUrl)

		assert.Nil(t, actual)
		assert.EqualError(t, err, "mapping not found")
	})

	t.Run("shoduld return replacer wihout error", func(t *testing.T) {
		parsedUrl, _ := url.Parse("http://localhost")

		actual, err := factory.Make(parsedUrl)

		assert.NotNil(t, actual)
		assert.NoError(t, err)
	})

	t.Run("shoduld return error when requst sheme and mapping sheme not equal", func(t *testing.T) {
		parsedUrl, _ := url.Parse("https://localhost")

		actual, err := factory.Make(parsedUrl)

		assert.Nil(t, actual)
		assert.EqualError(t, err, "mapping not found")
	})
}