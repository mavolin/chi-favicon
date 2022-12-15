package favicon

import (
	"bytes"
	"encoding/json"
	"image"
	"net/http"

	"github.com/disintegration/imaging"
	"github.com/go-chi/chi/v5"
)

type Options struct {
	// Name is the name of the app as used for the webmanifest.
	Name string
	// ShortName is the short name of the app as used for the webmanifest.
	//
	// If empty, [Options.Name] will be used.
	ShortName string
	// The display mode as used in the webmanifest.
	//
	// Defaults to `standalone`.
	Display string
	// StartURL is the start URL as used in the webmanifest.
	StartURL string
	// ThemeColor is the theme color as used in the webmanifest.
	//
	// Defaults to `#ffffff`.
	ThemeColor string
	// BackgroundColor is the background color as used in the webmanifest.
	//
	// Defaults to `#ffffff`.
	BackgroundColor string

	// TileColor is the tile color as used in the browserconfig.
	//
	// Defaults to `#da532c`.
	TileColor string

	// Favicon is png-encoded base icon.
	Favicon []byte
	// AppleTouchIcon is an alternative icon used as the apple-touch-icon.
	//
	// If not set, [Options.Favicon] is used.
	AppleTouchIcon []byte
}

// Add generates the favicons and adds the following routes to the passed
// [chi.Router].
// It assumes the router represents the root of the domain.
//
//   - apple-touch-icon.png
//   - favicon.png
//   - favicon-32x32.png
//   - favicon-16x16.png
//   - browserconfig.xml
//   - mstile-150x150.png
//   - site.webmanifest
//   - android-chrome-512x512.png
//   - android-chrome-192x192.png
func Add(r chi.Router, o Options) error {
	faviconImg, err := imaging.Decode(bytes.NewReader(o.Favicon))
	if err != nil {
		return err
	}

	appleTouchIconImg := faviconImg
	if o.AppleTouchIcon != nil {
		appleTouchIconImg, err = imaging.Decode(bytes.NewReader(o.AppleTouchIcon))
		if err != nil {
			return err
		}
	}

	if err = addAppleTouchIcon(r, appleTouchIconImg); err != nil {
		return err
	}

	if err = addFavicon(r, faviconImg); err != nil {
		return err
	}

	if err = addWebmanifest(r, faviconImg, o); err != nil {
		return err
	}

	return addBrowserConfig(r, faviconImg, o.TileColor)
}

func addAppleTouchIcon(r chi.Router, img image.Image) error {
	return addIcon(r, addIconOptions{
		name:   "apple-touch-icon.png",
		img:    img,
		size:   180,
		format: imaging.PNG,
		mime:   "image/png",
	})
}

func addFavicon(r chi.Router, img image.Image) error {
	err := addIcon(r, addIconOptions{
		name:   "favicon.png",
		img:    img,
		size:   48,
		format: imaging.PNG,
		mime:   "image/png",
	})
	if err != nil {
		return err
	}

	err = addIcon(r, addIconOptions{
		name:   "favicon-32x32.png",
		img:    img,
		size:   32,
		format: imaging.PNG,
		mime:   "image/png",
	})
	if err != nil {
		return err
	}

	return addIcon(r, addIconOptions{
		name:   "favicon-16x16.png",
		img:    img,
		size:   16,
		format: imaging.PNG,
		mime:   "image/png",
	})
}

type (
	webmanifest struct {
		Name            string            `json:"name"`
		ShortName       string            `json:"short_name"`
		Display         string            `json:"display"`
		StartURL        string            `json:"start_url,omitempty"`
		BackgroundColor string            `json:"background_color"`
		ThemeColor      string            `json:"theme_color"`
		Icons           []webmanifestIcon `json:"icons"`
	}

	webmanifestIcon struct {
		Src   string `json:"src"`
		Sizes string `json:"sizes"`
		Type  string `json:"type"`
	}
)

func addWebmanifest(r chi.Router, img image.Image, o Options) error {
	manifest := webmanifest{
		Name:      o.Name,
		ShortName: o.ShortName,
		Display:   o.Display,
		StartURL:  o.StartURL,
		Icons: []webmanifestIcon{
			{
				Src:   "/android-chrome-192x192.png",
				Sizes: "192x192",
				Type:  "image/png",
			},
			{
				Src:   "/android-chrome-512x512.png",
				Sizes: "512x512",
				Type:  "image/png",
			},
		},
	}

	if manifest.Display == "" {
		manifest.Display = "standalone"
	}

	if manifest.ThemeColor == "" {
		manifest.ThemeColor = "#ffffff"
	}

	if manifest.BackgroundColor == "" {
		manifest.BackgroundColor = "#ffffff"
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	r.Get("/site.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		_, _ = w.Write(manifestJSON)
	})

	err = addIcon(r, addIconOptions{
		name:   "android-chrome-192x192.png",
		img:    img,
		size:   192,
		format: imaging.PNG,
		mime:   "image/png",
	})
	if err != nil {
		return err
	}

	return addIcon(r, addIconOptions{
		name:   "android-chrome-512x512.png",
		img:    img,
		size:   512,
		format: imaging.PNG,
		mime:   "image/png",
	})
}

func addBrowserConfig(r chi.Router, img image.Image, tileColor string) error {
	if tileColor == "" {
		tileColor = "#da532c"
	}

	browserConfig := []byte(`<?xml version="1.0" encoding="utf-8"?>
<browserconfig>
    <msapplication>
        <tile>
            <square150x150logo src="/mstile-150x150.png"/>
            <TileColor>` + tileColor + `</TileColor>
        </tile>
    </msapplication>
</browserconfig>`)

	r.Get("/browserconfig.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(browserConfig)
	})

	return addIcon(r, addIconOptions{
		name:   "mstile-150x150.png",
		img:    img,
		size:   150,
		format: imaging.PNG,
		mime:   "image/png",
	})
}

type addIconOptions struct {
	name   string
	img    image.Image
	size   int
	format imaging.Format
	mime   string
}

func addIcon(r chi.Router, o addIconOptions) error {
	ico := imaging.Resize(o.img, o.size, o.size, imaging.Lanczos)

	var buf bytes.Buffer
	err := imaging.Encode(&buf, ico, o.format)
	if err != nil {
		return err
	}

	data := buf.Bytes()

	r.Get("/"+o.name, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", o.mime)
		_, _ = w.Write(data)
	})

	return nil
}
