// Package file provides HTTP handlers for file-related operations.
package file

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FileController handles file related endpoints
type FileController struct {
	DB      *database.DBinstanceStruct
	Storage StorageClient
}

const (
	resumeObjectPrefix = "resumes"
	logoObjectPrefix   = "logos"
	bannerObjectPrefix = "banners"
)

// NewFileController creates a new instance of FileController
func NewFileController(db *database.DBinstanceStruct, storage StorageClient) *FileController {
	return &FileController{
		DB:      db,
		Storage: storage,
	}
}

// UploadResume function handles the process of uploading a resume file for a user and updating the
// user's information in the database.
// @Summary Upload resume file for CPSK
// @Description Only file that smaller than 10 MB with .pdf extension is permitted
// @Tags CPSK
// @Accept mpfd
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param resume formData file true "Upload your resume file"
// @Success 200 {object} model.CPSKUser "Successfully upload resume"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as CPSK, User is banned"
// @Failure 413 {object} utilities.ErrorResponse "File size is larger than 10 MB"
// @Failure 415 {object} utilities.ErrorResponse "File extension is not allowed"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /cpsk/profile/resume [post]
func (jc *FileController) UploadResume(c *gin.Context) {

	var cpskUser = model.CPSKUser{}

	user, err := utilities.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{Error: err.Error()})
		return
	}

	// Retrieve original profile from DB
	if err := jc.DB.Preload("User").Where("user_id = ?", user.ID.String()).First(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
		return
	}

	rawFile, err := c.FormFile("resume")
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		c.JSON(http.StatusRequestEntityTooLarge, utilities.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve file: %s", err.Error()),
		})
		return
	}

	extension := strings.ToLower(filepath.Ext(rawFile.Filename))
	if extension != ".pdf" {
		c.JSON(http.StatusUnsupportedMediaType, utilities.ErrorResponse{
			Error: fmt.Sprintf("Unsupported file extension: %s", extension),
		})
		return
	}

	f, err := rawFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{Error: "Cannot open file"})
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal("Failed to close file")
		}
	}()

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{Error: "Cannot read file"})
		return
	}

	if err := jc.persistFileData(&cpskUser.Resume, fileBytes, ".pdf", resumeObjectPrefix); err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to store resume: %s", err.Error()),
		})
		return
	}

	if err := jc.DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, cpskUser)
}

// companyUpload function handles process of reading files from company upload.
func (jc *FileController) companyUpload(c *gin.Context, fName string) (model.CompanyUser, []byte, string) {
	var company = model.CompanyUser{}

	user, err := utilities.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{Error: err.Error()})
		return company, nil, ""
	}

	// Retrieve original profile from DB
	if err := jc.DB.Preload("User").Where("user_id = ?", user.ID.String()).First(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
		return company, nil, ""
	}

	rawFile, err := c.FormFile(fName)
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		c.JSON(http.StatusRequestEntityTooLarge, utilities.ErrorResponse{
			Error: err.Error(),
		})
		return company, nil, ""
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve file: %s", err.Error()),
		})
		return company, nil, ""
	}

	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	extension := strings.ToLower(filepath.Ext(rawFile.Filename))

	if !allowedExtensions[extension] {
		c.JSON(http.StatusUnsupportedMediaType, utilities.ErrorResponse{
			Error: fmt.Sprintf("Unsupported file extension: %s", extension),
		})
		return company, nil, ""
	}

	f, err := rawFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{Error: "Cannot open file"})
		return company, nil, ""
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal("Failed to close file")
		}
	}()

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{Error: "Cannot read file"})
		return company, nil, ""
	}

	return company, fileBytes, extension
}

// UploadLogo function handles company's logo uploading and updating company profile in database.
// @Summary Upload logo file for company
// @Description Only file that smaller than 10 MB with .jpg, .jpeg, or .png extension is permitted
// @Tags Company
// @Accept mpfd
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param logo formData file true "Upload your logo file"
// @Success 200 {object} model.CompanyUser "Successfully upload logo"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as company, User is banned"
// @Failure 413 {object} utilities.ErrorResponse "File size is larger than 10 MB"
// @Failure 415 {object} utilities.ErrorResponse "File extension is not allowed"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /company/profile/logo [post]
func (jc *FileController) UploadLogo(c *gin.Context) {

	company, fileBytes, fileExtension := jc.companyUpload(c, "logo")

	if fileBytes == nil {
		return
	}

	if err := jc.persistFileData(&company.Logo, fileBytes, fileExtension, logoObjectPrefix); err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to store logo: %s", err.Error()),
		})
		return
	}

	if err := jc.DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

// UploadBanner function handles company's banner uploading and updating company profile in database.
// @Summary Upload banner file for company
// @Description Only file that smaller than 10 MB with .jpg, .jpeg, or .png extension is permitted
// @Tags Company
// @Accept mpfd
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param banner formData file true "Upload your banner file"
// @Success 200 {object} model.CompanyUser "Successfully upload banner"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as company, User is banned"
// @Failure 413 {object} utilities.ErrorResponse "File size is larger than 10 MB"
// @Failure 415 {object} utilities.ErrorResponse "File extension is not allowed"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /company/profile/banner [post]
func (jc *FileController) UploadBanner(c *gin.Context) {
	company, fileBytes, fileExtension := jc.companyUpload(c, "banner")

	if fileBytes == nil {
		return
	}

	if err := jc.persistFileData(&company.Banner, fileBytes, fileExtension, bannerObjectPrefix); err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to store banner: %s", err.Error()),
		})
		return
	}

	if err := jc.DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

// GetFile function retrieves a file from the database and sends it as a downloadable attachment in
// the response.
// @Summary Retrieve dowloadable attachment
// @Tags File
// @Produce octet-stream
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param id path string true "ID of wanted file"
// @Success 200 {string} binary "Successfully retrieve file"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User is banned"
// @Failure 404 {object} utilities.ErrorResponse "Given file id not found"
// @Failure 500 {object} utilities.ErrorResponse "Fail to send file content"
// @Router /file/{id} [get]
func (jc *FileController) GetFile(c *gin.Context) {
	var file model.File
	id := c.Param("id")

	if err := jc.DB.First(&file, id).Error; err != nil {
		c.String(http.StatusNotFound, "File not found")
		return
	}

	jc.writeFileResponse(c, &file)
}

func (jc *FileController) writeFileResponse(c *gin.Context, file *model.File) {
	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+fmt.Sprint(file.ID)+file.Extension)
	c.Writer.Header().Set("Content-Type", "application/octet-stream")

	if file.StorageObjectName != nil {
		if jc.Storage == nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: "Cloud storage is disabled while the requested file is stored remotely",
			})
			return
		}
		reader, size, err := jc.Storage.DownloadFile(*file.StorageObjectName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to download file from storage: %s", err.Error()),
			})
			return
		}
		defer func() {
			if err := reader.Close(); err != nil {
				log.Printf("failed to close storage reader: %v", err)
			}
		}()

		if size > 0 {
			c.Writer.Header().Set("Content-Length", fmt.Sprint(size))
		}
		if _, err := io.Copy(c.Writer, reader); err != nil {
			jc.handleWriterError(c, err)
		}
		return
	}

	c.Writer.Header().Set("Content-Length", fmt.Sprint(len(file.Content)))
	if _, err := c.Writer.Write(file.Content); err != nil {
		jc.handleWriterError(c, err)
	}
}

func (jc *FileController) handleWriterError(c *gin.Context, err error) {
	if !c.Writer.Written() {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: "Failed to send file content",
		})
	} else {
		c.Abort()
	}
}

func (jc *FileController) persistFileData(file *model.File, fileBytes []byte, extension, prefix string) error {
	file.Extension = extension
	if jc.Storage == nil {
		file.Content = fileBytes
		file.StorageObjectName = nil
		return nil
	}

	objectName := fmt.Sprintf("%s/%s%s", prefix, uuid.NewString(), extension)
	if err := jc.Storage.UploadFile(objectName, bytes.NewReader(fileBytes)); err != nil {
		return err
	}

	file.StorageObjectName = &objectName
	file.Content = nil
	return nil
}
