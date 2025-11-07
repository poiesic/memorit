package memorit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabase(t *testing.T) {
	t.Run("create new database", func(t *testing.T) {
		tmpDir := filepath.Join(t.TempDir(), "test_db")
		db, err := NewDatabase(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		// Verify components are initialized
		assert.NotNil(t, db.ChatRepository())
		assert.NotNil(t, db.ConceptRepository())
		assert.NotNil(t, db.backend)
		assert.NotNil(t, db.logger)
	})

	t.Run("error with invalid path", func(t *testing.T) {
		// Try to create a database at a file path instead of directory
		tmpFile := filepath.Join(t.TempDir(), "not_a_dir")
		err := os.WriteFile(tmpFile, []byte("test"), 0644)
		require.NoError(t, err)

		db, err := NewDatabase(tmpFile)
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}

func TestDatabase_Close(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDatabase(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Close the database
	err = db.Close()
	assert.NoError(t, err)
}

func TestDatabase_FactoryMethods(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDatabase(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	t.Run("can create ingestion pipeline", func(t *testing.T) {
		pipeline, err := db.NewIngestionPipeline()
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		pipeline.Release()
	})

	t.Run("can create searcher", func(t *testing.T) {
		searcher, err := db.NewSearcher()
		require.NoError(t, err)
		require.NotNil(t, searcher)
	})
}
