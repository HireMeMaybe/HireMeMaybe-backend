package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"HireMeMaybe-backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestPersistFileData_UsesCloudStorage(t *testing.T) {
	mockStorage := newMockStorageClient()
	ctrl := NewFileController(nil, mockStorage)
	file := &model.File{}
	data := []byte("hello world")

	err := ctrl.persistFileData(file, data, ".pdf", resumeObjectPrefix)
	require.NoError(t, err)

	require.NotNil(t, file.StorageObjectName)
	require.True(t, strings.HasPrefix(*file.StorageObjectName, resumeObjectPrefix+"/"))
	require.Nil(t, file.Content)
	require.Equal(t, ".pdf", file.Extension)
	require.Contains(t, mockStorage.uploaded, *file.StorageObjectName)
	require.Equal(t, data, mockStorage.uploaded[*file.StorageObjectName])
}

func TestPersistFileData_FallsBackToDatabase(t *testing.T) {
	ctrl := NewFileController(nil, nil)
	file := &model.File{}
	data := []byte("legacy")

	err := ctrl.persistFileData(file, data, ".png", logoObjectPrefix)
	require.NoError(t, err)

	require.Nil(t, file.StorageObjectName)
	require.Equal(t, data, file.Content)
	require.Equal(t, ".png", file.Extension)
}

func TestPersistFileData_UploadError(t *testing.T) {
	mockStorage := newMockStorageClient()
	mockStorage.uploadErr = errors.New("boom")
	ctrl := NewFileController(nil, mockStorage)
	file := &model.File{}

	err := ctrl.persistFileData(file, []byte("fail"), ".pdf", resumeObjectPrefix)
	require.Error(t, err)
	require.EqualError(t, err, "boom")
}

func TestWriteFileResponse_CloudStorage(t *testing.T) {
	mockStorage := newMockStorageClient()
	mockStorage.downloadPayload["resumes/foo"] = []byte("downloaded")
	ctrl := NewFileController(nil, mockStorage)
	objectName := "resumes/foo"
	file := &model.File{ID: 42, Extension: ".pdf", StorageObjectName: &objectName}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ctrl.writeFileResponse(c, file)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "downloaded", w.Body.String())
	require.Equal(t, "attachment; filename=42.pdf", w.Header().Get("Content-Disposition"))
	require.Equal(t, fmt.Sprint(len("downloaded")), w.Header().Get("Content-Length"))
}

func TestWriteFileResponse_LegacyContent(t *testing.T) {
	ctrl := NewFileController(nil, nil)
	legacyContent := []byte("legacy")
	file := &model.File{ID: 7, Extension: ".jpg", Content: legacyContent}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ctrl.writeFileResponse(c, file)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, legacyContent, w.Body.Bytes())
	require.Equal(t, fmt.Sprint(len(legacyContent)), w.Header().Get("Content-Length"))
}

func TestWriteFileResponse_RemoteButStorageDisabled(t *testing.T) {
	ctrl := NewFileController(nil, nil)
	objectName := "logos/foo"
	file := &model.File{ID: 8, Extension: ".png", StorageObjectName: &objectName}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ctrl.writeFileResponse(c, file)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "Cloud storage is disabled")
}

type mockStorageClient struct {
	uploaded        map[string][]byte
	downloadPayload map[string][]byte
	uploadErr       error
	downloadErr     error
}

func newMockStorageClient() *mockStorageClient {
	return &mockStorageClient{
		uploaded:        make(map[string][]byte),
		downloadPayload: make(map[string][]byte),
	}
}

func (m *mockStorageClient) UploadFile(objectName string, fileData io.Reader) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	buf, err := io.ReadAll(fileData)
	if err != nil {
		return err
	}
	m.uploaded[objectName] = buf
	return nil
}

func (m *mockStorageClient) DownloadFile(objectName string) (io.ReadCloser, int64, error) {
	if m.downloadErr != nil {
		return nil, 0, m.downloadErr
	}
	data, ok := m.downloadPayload[objectName]
	if !ok {
		return nil, 0, fmt.Errorf("object %s not found", objectName)
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}
