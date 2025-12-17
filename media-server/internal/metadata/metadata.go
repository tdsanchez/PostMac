package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwcarlsen/goexif/exif"

	"github.com/tdsanchez/PostMac/media-server/internal/models"
)

// GetFileMetadata extracts metadata and EXIF information from a file
func GetFileMetadata(path string) (*models.FileMetadata, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	metadata := &models.FileMetadata{
		FileName: info.Name(),
		FileSize: info.Size(),
		Modified: info.ModTime(),
	}

	metadata.Created = info.ModTime()

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".tif" || ext == ".tiff" {
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()

			x, err := exif.Decode(file)
			if err == nil {
				if tag, err := x.Get(exif.PixelXDimension); err == nil {
					if val, err := tag.Int(0); err == nil {
						metadata.Width = val
					}
				}
				if tag, err := x.Get(exif.PixelYDimension); err == nil {
					if val, err := tag.Int(0); err == nil {
						metadata.Height = val
					}
				}

				if tag, err := x.Get(exif.Make); err == nil {
					metadata.Make, _ = tag.StringVal()
				}
				if tag, err := x.Get(exif.Model); err == nil {
					metadata.Model, _ = tag.StringVal()
				}

				if tag, err := x.Get(exif.DateTime); err == nil {
					metadata.DateTime, _ = tag.StringVal()
				} else if tag, err := x.Get(exif.DateTimeOriginal); err == nil {
					metadata.DateTime, _ = tag.StringVal()
				}

				if tag, err := x.Get(exif.Orientation); err == nil {
					if val, err := tag.Int(0); err == nil {
						orientations := map[int]string{
							1: "Normal", 2: "Flipped Horizontal", 3: "Rotated 180°",
							4: "Flipped Vertical", 5: "Rotated 90° CCW, Flipped",
							6: "Rotated 90° CW", 7: "Rotated 90° CW, Flipped",
							8: "Rotated 90° CCW",
						}
						metadata.Orientation = orientations[val]
					}
				}

				if tag, err := x.Get(exif.ISOSpeedRatings); err == nil {
					if val, err := tag.Int(0); err == nil {
						metadata.ISO = fmt.Sprintf("ISO %d", val)
					}
				}

				if tag, err := x.Get(exif.FNumber); err == nil {
					if num, denom, err := tag.Rat2(0); err == nil && denom != 0 {
						metadata.FNumber = fmt.Sprintf("f/%.1f", float64(num)/float64(denom))
					}
				}

				if tag, err := x.Get(exif.ExposureTime); err == nil {
					if num, denom, err := tag.Rat2(0); err == nil && denom != 0 {
						if num < denom {
							metadata.ExposureTime = fmt.Sprintf("1/%d sec", denom/num)
						} else {
							metadata.ExposureTime = fmt.Sprintf("%.1f sec", float64(num)/float64(denom))
						}
					}
				}

				if tag, err := x.Get(exif.FocalLength); err == nil {
					if num, denom, err := tag.Rat2(0); err == nil && denom != 0 {
						metadata.FocalLength = fmt.Sprintf("%.1f mm", float64(num)/float64(denom))
					}
				}

				if tag, err := x.Get(exif.Flash); err == nil {
					if val, err := tag.Int(0); err == nil {
						if val&1 == 1 {
							metadata.Flash = "Fired"
						} else {
							metadata.Flash = "Not Fired"
						}
					}
				}

				if tag, err := x.Get(exif.WhiteBalance); err == nil {
					if val, err := tag.Int(0); err == nil {
						if val == 0 {
							metadata.WhiteBalance = "Auto"
						} else {
							metadata.WhiteBalance = "Manual"
						}
					}
				}

				if tag, err := x.Get(exif.Artist); err == nil {
					metadata.Artist, _ = tag.StringVal()
				}
				if tag, err := x.Get(exif.Copyright); err == nil {
					metadata.Copyright, _ = tag.StringVal()
				}

				if tag, err := x.Get(exif.Software); err == nil {
					metadata.Software, _ = tag.StringVal()
				}
			}
		}
	}

	return metadata, nil
}
