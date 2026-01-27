package main

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ImageInfo holds information about an image file
type ImageInfo struct {
	Name      string
	Size      string
	CreatedAt time.Time
	URL       string
}

var allowedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
}

func isAllowedImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return allowedImageExts[ext]
}

func safeUploadPath(filename string) (string, bool) {
	if filename == "" {
		return "", false
	}
	base := filepath.Base(filename)
	if base != filename {
		return "", false
	}
	if strings.ContainsAny(filename, `/\`) {
		return "", false
	}
	if !isAllowedImage(filename) {
		return "", false
	}
	return filepath.Join("./uploads", base), true
}

// GetImages retrieves images from the uploads directory with pagination
func GetImages(page int, perPage int) ([]ImageInfo, int, error) {
	uploadsDir := "./uploads"

	// DEBUG: Log directory read attempt
	println("[DEBUG] Attempting to read directory:", uploadsDir)

	// Read all files in the uploads directory
	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		println("[DEBUG] Error reading directory:", err.Error())
		return nil, 0, err
	}

	// DEBUG: Log number of entries found
	println("[DEBUG] Found", len(entries), "entries in directory")

	var images []ImageInfo

	// Collect image files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's an image file by extension
		if isAllowedImage(entry.Name()) {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Get file size in human-readable format
			size := info.Size()
			sizeStr := formatFileSize(size)

			images = append(images, ImageInfo{
				Name:      entry.Name(),
				Size:      sizeStr,
				CreatedAt: info.ModTime(),
				URL:       "/uploads/" + entry.Name(),
			})
		}
	}

	// Sort by creation time in descending order
	sort.Slice(images, func(i, j int) bool {
		return images[i].CreatedAt.After(images[j].CreatedAt)
	})

	// DEBUG: Log pagination calculation
	total := len(images)
	println("[DEBUG] Total images found:", total)

	totalPages := (total + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	// DEBUG: Log pagination details
	println("[DEBUG] Total pages:", totalPages, "Requested page:", page)

	// Adjust page number if out of range
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Get the slice for the current page
	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}

	// DEBUG: Log slice calculation
	println("[DEBUG] Slice range: start =", start, ", end =", end, ", total =", total)

	var pagedImages []ImageInfo
	if start < total {
		pagedImages = images[start:end]
	}

	// DEBUG: Log result
	println("[DEBUG] Returning pagedImages, is nil:", pagedImages == nil, ", length:", len(pagedImages))

	return pagedImages, totalPages, nil
}

// formatFileSize converts file size to human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(bytes)/float64(div), 'f', 2, 64) + " KMGTPE"[exp:exp+1] + "B"
}

func InitRouter() *gin.Engine {
	r := gin.Default()

	// Create template with custom functions
	r.SetFuncMap(template.FuncMap{
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
		"sequence": func(start, end int) []int {
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
	})

	// Load HTML templates
	r.LoadHTMLGlob("views/*")

	// Serve static files from uploads directory
	r.Static("/uploads", "./uploads")

	// Index page with image list
	r.GET("/", func(c *gin.Context) {
		// Get page parameter, default to 1
		pageStr := c.DefaultQuery("page", "1")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		// Get images for current page
		images, totalPages, err := GetImages(page, 20)
		if err != nil {
			println("[DEBUG] GetImages returned error:", err.Error())
			c.HTML(http.StatusInternalServerError, "index.html", gin.H{
				"error": "Failed to load images",
			})
			return
		}

		// DEBUG: Log what we're passing to template
		println("[DEBUG] Passing to template - images is nil:", images == nil, ", len:", len(images), ", totalPages:", totalPages)

		c.HTML(http.StatusOK, "index.html", gin.H{
			"images":      images,
			"currentPage": page,
			"totalPages":  totalPages,
		})
	})

	r.POST("/api/v1/upload", func(c *gin.Context) {
		// Get image
		file, err := c.FormFile("image")

		if err != nil {
			c.JSON(400, gin.H{"error": "No image is received"})
			return
		}

		// Generate filename using nanosecond timestamp
		ext := filepath.Ext(file.Filename)
		timestamp := time.Now().UnixNano()
		newFilename := strconv.FormatInt(timestamp, 10) + ext

		// Save the uploaded file to a specific location
		if err := c.SaveUploadedFile(file, "./uploads/"+newFilename); err != nil {
			c.JSON(500, gin.H{"error": "Unable to save the image"})
			return
		}
		c.JSON(200, gin.H{"message": "Image uploaded successfully", "filename": newFilename})
	})

	r.POST("/api/v1/delete", func(c *gin.Context) {
		var body struct {
			Filename string `json:"filename"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		targetPath, ok := safeUploadPath(body.Filename)
		if !ok {
			c.JSON(400, gin.H{"error": "Invalid filename"})
			return
		}

		if err := os.Remove(targetPath); err != nil {
			if os.IsNotExist(err) {
				c.JSON(404, gin.H{"error": "File not found"})
				return
			}
			c.JSON(500, gin.H{"error": "Failed to delete file"})
			return
		}

		c.JSON(200, gin.H{"message": "Image deleted successfully", "filename": body.Filename})
	})

	r.POST("/api/v1/delete-batch", func(c *gin.Context) {
		var body struct {
			Filenames []string `json:"filenames"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}
		if len(body.Filenames) == 0 {
			c.JSON(400, gin.H{"error": "No filenames provided"})
			return
		}

		deleted := make([]string, 0, len(body.Filenames))
		failed := make(map[string]string)

		for _, name := range body.Filenames {
			targetPath, ok := safeUploadPath(name)
			if !ok {
				failed[name] = "invalid filename"
				continue
			}

			if err := os.Remove(targetPath); err != nil {
				if os.IsNotExist(err) {
					failed[name] = "not found"
				} else {
					failed[name] = "delete failed"
				}
				continue
			}

			deleted = append(deleted, name)
		}

		c.JSON(200, gin.H{
			"deleted": deleted,
			"failed":  failed,
		})
	})

	return r
}

func main() {
	r := InitRouter()
	r.Run(":38889")
}
