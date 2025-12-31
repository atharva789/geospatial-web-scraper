# IndexFTP and Metadata Extraction - Key Changes Summary

## Overview
This document summarizes the major architectural changes to the FTP indexing system in `crawler2.go` and related metadata structures in `structs.go`.

---

## 1. IndexFTP Function Changes

### Previous Behavior
- Returned `FTPDirectory` (value type)
- Flattened all discovered files into a single directory
- Attempted BFS traversal but mixed HTML parsing with HTTP fetching
- No proper tree hierarchy preservation
- Mixed parent directory navigation with actual subdirectories

### Current Behavior (Lines 168-253 in crawler2.go)

**Signature Change:**
```go
// Old: func IndexFTP(n *html.Node, resp *http.Response) (FTPDir, error)
// New: func IndexFTP(n *html.Node, resp *http.Response) (*FTPDirectory, error)
```

**Key Improvements:**

1. **Returns Pointer Instead of Value**
   - Changed from `FTPDirectory` to `*FTPDirectory`
   - Enables proper parent-child pointer relationships
   - More memory efficient for large trees

2. **Hierarchical Tree Structure**
   - Delegates to `indexFTPRecursive()` helper function
   - Recursively builds tree that mirrors actual FTP directory structure
   - Each `FTPDirectory` has:
     - `Parent *FTPDirectory` - pointer to parent directory
     - `SubDirectories []*FTPDirectory` - pointers to child directories
     - `DownloadFiles []GeoFile` - files in current directory only

3. **Proper Subdirectory Handling**
   - Skips navigation links (`../`, `..`, `/`)
   - Only follows actual subdirectory links
   - Fetches and parses each subdirectory's HTML
   - Maintains parent-child relationships via pointers

4. **Smart Metadata Matching**
   - `matchGeoFilesWithMetadata()` function (lines 257-280)
   - Pairs geo files with metadata by matching base filenames
   - Example: `data.tif` matches with `data.xml`
   - Each `GeoFile` has both `URL` and `Metadata` fields populated

---

## 2. New IndexS3 Function

**Purpose:** Handle S3 buckets that use JavaScript-based directory browsers

**Location:** Lines 283-407 in crawler2.go

**How It Works:**
1. Parses S3 URL formats (both `index.html?prefix=` and direct paths)
2. Uses S3 ListObjectsV2 API instead of HTML parsing
3. Handles pagination for large buckets (`IsTruncated`, `NextContinuationToken`)
4. Builds same hierarchical `*FTPDirectory` structure as IndexFTP
5. Recursively processes subdirectories (S3 "common prefixes")

**New Types Added (Lines 14-35):**
- `S3ListBucketResult` - XML response structure
- `S3Object` - represents individual S3 objects
- `S3Prefix` - represents subdirectory prefixes

---

## 3. DatasetMetadata Structure Changes

### Location: `structs.go` lines 159-231

### New Fields Added:

**Geographic Bounds:**
```go
Bounds BoundingBox `json:"bounds,omitempty"`
```
- `EastBC`, `WestBC`, `NorthBC`, `SouthBC` (float64)
- Decimal degrees format

**Horizontal Metadata:**
```go
HorizontalMeta HorizontalMeta `json:"horizontalmeta,omitempty"`
```
- `LatRes`, `LongRes` (float64) - resolution in lat/long
- `GeoUnit` (string) - units (e.g., "degrees", "meters")
- `HorizontalCRS` (string) - coordinate reference system

**Vertical Metadata:**
```go
VerticalMeta VerticalMeta `json:"verticalmeta,omitempty"`
```
- `VerticalCRS` (string) - vertical coordinate system
- `AltRes` (float64) - altitude resolution
- `AltUnits` (string) - altitude units (e.g., "meters", "feet")

**Temporal Coverage:**
```go
StartDate string `json:"start_date,omitempty"` // ISO 8601: YYYY-MM-DD
EndDate   string `json:"end_date,omitempty"`   // ISO 8601: YYYY-MM-DD
```

### Breaking Change:
- `Source` changed from `*DatasetMetadata` (recursive pointer) to `string`
- Now stores source/provider name directly instead of nested metadata object

---

## 4. FTPDirectory Structure

**Location:** `structs.go` lines 174-182

**Current Structure:**
```go
type FTPDirectory struct {
    Parent         *FTPDirectory   // Parent directory (nil for root)
    SubDirectories []*FTPDirectory // Child directories
    DownloadFiles  []GeoFile       // Geospatial files with metadata
}
```

**GeoFile Structure:**
```go
type GeoFile struct {
    URL      string // Download URL for the geospatial file
    Metadata string // Optional URL to metadata/sidecar file
}
```

**Key Properties:**
- Tree structure with proper parent-child pointers
- Files stored only in their actual directory (not flattened)
- Metadata URLs paired with data URLs
- Empty directories automatically pruned (no files = not added to tree)

---

## 5. Usage Example

```go
// For traditional FTP listings
doc, resp, err := getDOMFromURL("https://example.com/ftp/")
tree, err := IndexFTP(doc, resp)

// For S3 buckets
tree, err := IndexS3("https://prd-tnm.s3.amazonaws.com/index.html?prefix=StagedProducts/")

// Both return *FTPDirectory with same structure
// Access files: tree.DownloadFiles
// Navigate: tree.SubDirectories[0].DownloadFiles
// Check parent: tree.SubDirectories[0].Parent == tree
```

---

## 6. Testing Changes

**Test File:** `cmd/test_indexftp/main.go`

- Auto-detects S3 URLs via `IsS3URL()` helper
- Routes to `IndexS3` or `IndexFTP` appropriately
- Updated to handle pointer return type (`*FTPDirectory`)
- Added `TestIndexS3()` function in `test_indexftp.go`

---

## 7. Migration Notes

### If You Were Using Old IndexFTP:

**Before:**
```go
result, err := IndexFTP(doc, resp)
files := result.DownloadFiles  // All files flattened
```

**After:**
```go
result, err := IndexFTP(doc, resp)
files := result.DownloadFiles  // Only files in root directory
// Access subdirectory files:
for _, subdir := range result.SubDirectories {
    subFiles := subdir.DownloadFiles
}
```

### Recursively Collect All Files:
```go
func CollectAllFiles(dir *FTPDirectory) []GeoFile {
    var files []GeoFile
    files = append(files, dir.DownloadFiles...)
    for _, subdir := range dir.SubDirectories {
        files = append(files, CollectAllFiles(subdir)...)
    }
    return files
}
```

---

## 8. Important Implementation Details

1. **Domain Filtering:** IndexFTP now skips external navigation links that would cause infinite loops or fetch unrelated pages

2. **Metadata Matching Algorithm:**
   - Extracts base filename (without extension)
   - Builds map of `basename -> metadata_url`
   - Matches geo files to metadata by basename
   - Handles multiple geo files with same basename in different subdirectories

3. **S3 Pagination:**
   - Uses `delimiter=/` to get hierarchical listing
   - Follows `IsTruncated` flag for continuation
   - Properly handles large buckets with 1000+ objects per prefix

4. **Memory Efficiency:**
   - Pointer-based tree prevents duplication
   - Prunes empty directories automatically
   - Closes HTTP response bodies immediately in loops

---

## 9. File Locations

- **IndexFTP implementation:** `internal/services/crawler_service/internal/crawler/crawler2.go:168-280`
- **IndexS3 implementation:** `internal/services/crawler_service/internal/crawler/crawler2.go:283-407`
- **FTPDirectory struct:** `internal/services/crawler_service/internal/crawler/structs.go:174-182`
- **DatasetMetadata struct:** `internal/services/crawler_service/internal/crawler/structs.go:159-231`
- **Test harness:** `internal/services/crawler_service/cmd/test_indexftp/main.go`

---

## 10. Backward Compatibility

**Breaking Changes:**
- Return type changed to pointer
- Tree structure instead of flat list
- `DatasetMetadata.Source` is now string instead of `*DatasetMetadata`

**Type Aliases (for compatibility):**
- `FTPDir = FTPDirectory`
- `downloadMetadata = DatasetMetadata`
