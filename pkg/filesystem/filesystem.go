package filesystem

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// FileInfo 表示文件信息
type FileInfo struct {
	Name        string
	Path        string
	Size        int64
	Mode        fs.FileMode
	ModTime     time.Time
	IsDir       bool
	IsSymlink   bool
	Owner       string
	Group       string
	Checksum    map[string]string
	Permissions string
	ContentType string
}

// FileOperation 表示文件操作结果
type FileOperation struct {
	Success bool
	Message string
	Error   error
}

// ListFiles 列出目录内容
func ListFiles(path string, recursive bool, showHidden bool) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏文件（如果需要）
		if !showHidden && strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 如果不是递归模式，只处理当前目录
		if !recursive && filepath.Dir(path) != filepath.Clean(path) {
			return nil
		}

		fileInfo := FileInfo{
			Name:    info.Name(),
			Path:    path,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		// 获取符号链接信息
		if info.Mode()&fs.ModeSymlink != 0 {
			fileInfo.IsSymlink = true
			if target, err := os.Readlink(path); err == nil {
				fileInfo.Path = fmt.Sprintf("%s -> %s", path, target)
			}
		}

		// 获取文件所有者信息
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			fileInfo.Owner = fmt.Sprintf("%d", stat.Uid)
			fileInfo.Group = fmt.Sprintf("%d", stat.Gid)
		}

		// 计算文件校验和
		if !info.IsDir() {
			fileInfo.Checksum = calculateChecksums(path)
		}

		// 获取文件权限
		fileInfo.Permissions = info.Mode().String()

		// 获取文件类型
		if !info.IsDir() {
			fileInfo.ContentType = detectContentType(path)
		}

		files = append(files, fileInfo)
		return nil
	})

	return files, err
}

// CopyFile 复制文件或目录
func CopyFile(src, dst string) FileOperation {
	info, err := os.Stat(src)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	if info.IsDir() {
		return copyDirectory(src, dst)
	}

	return copySingleFile(src, dst)
}

// MoveFile 移动文件或目录
func MoveFile(src, dst string) FileOperation {
	err := os.Rename(src, dst)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	return FileOperation{Success: true, Message: "文件移动成功"}
}

// DeleteFile 删除文件或目录
func DeleteFile(path string, recursive bool) FileOperation {
	info, err := os.Stat(path)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	if info.IsDir() && !recursive {
		return FileOperation{Success: false, Error: fmt.Errorf("目录非空，请使用递归删除")}
	}

	if info.IsDir() {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	return FileOperation{Success: true, Message: "文件删除成功"}
}

// CreateDirectory 创建目录
func CreateDirectory(path string, parents bool) FileOperation {
	var err error
	if parents {
		err = os.MkdirAll(path, 0755)
	} else {
		err = os.Mkdir(path, 0755)
	}

	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	return FileOperation{Success: true, Message: "目录创建成功"}
}

// ChangePermissions 修改文件权限
func ChangePermissions(path string, mode os.FileMode, recursive bool) FileOperation {
	if recursive {
		err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			return os.Chmod(path, mode)
		})
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	} else {
		err := os.Chmod(path, mode)
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	}
	return FileOperation{Success: true, Message: "权限修改成功"}
}

// ChangeOwner 修改文件所有者
func ChangeOwner(path string, uid, gid int, recursive bool) FileOperation {
	if recursive {
		err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			return os.Chown(path, uid, gid)
		})
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	} else {
		err := os.Chown(path, uid, gid)
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	}
	return FileOperation{Success: true, Message: "所有者修改成功"}
}

// CreateSymlink 创建符号链接
func CreateSymlink(target, link string) FileOperation {
	err := os.Symlink(target, link)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	return FileOperation{Success: true, Message: "符号链接创建成功"}
}

// Archive 创建归档文件
func Archive(files []string, output string, format string) FileOperation {
	switch format {
	case "zip":
		return createZipArchive(files, output)
	case "tar":
		return createTarArchive(files, output)
	case "tar.gz":
		return createTarGzArchive(files, output)
	default:
		return FileOperation{Success: false, Error: fmt.Errorf("不支持的归档格式: %s", format)}
	}
}

// Extract 解压归档文件
func Extract(archive, output string) FileOperation {
	switch {
	case strings.HasSuffix(archive, ".zip"):
		return extractZipArchive(archive, output)
	case strings.HasSuffix(archive, ".tar"):
		return extractTarArchive(archive, output)
	case strings.HasSuffix(archive, ".tar.gz") || strings.HasSuffix(archive, ".tgz"):
		return extractTarGzArchive(archive, output)
	default:
		return FileOperation{Success: false, Error: fmt.Errorf("不支持的归档格式")}
	}
}

// SearchFiles 搜索文件
func SearchFiles(root, pattern string, caseSensitive bool) ([]FileInfo, error) {
	var results []FileInfo

	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := info.Name()
		if !caseSensitive {
			name = strings.ToLower(name)
			pattern = strings.ToLower(pattern)
		}

		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return err
		}

		if matched {
			fileInfo := FileInfo{
				Name:    info.Name(),
				Path:    path,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime(),
				IsDir:   info.IsDir(),
			}
			results = append(results, fileInfo)
		}

		return nil
	})

	return results, err
}

// 辅助函数
func calculateChecksums(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	md5Hash := md5.New()
	sha1Hash := sha1.New()
	sha256Hash := sha256.New()

	writer := io.MultiWriter(md5Hash, sha1Hash, sha256Hash)
	if _, err := io.Copy(writer, file); err != nil {
		return nil
	}

	return map[string]string{
		"md5":    hex.EncodeToString(md5Hash.Sum(nil)),
		"sha1":   hex.EncodeToString(sha1Hash.Sum(nil)),
		"sha256": hex.EncodeToString(sha256Hash.Sum(nil)),
	}
}

func detectContentType(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "unknown"
	}

	return http.DetectContentType(buffer)
}

func copySingleFile(src, dst string) FileOperation {
	srcFile, err := os.Open(src)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	return FileOperation{Success: true, Message: "文件复制成功"}
}

func copyDirectory(src, dst string) FileOperation {
	err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copySingleFile(path, dstPath).Error
	})

	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	return FileOperation{Success: true, Message: "目录复制成功"}
}

func createZipArchive(files []string, output string) FileOperation {
	zipFile, err := os.Create(output)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		err := addFileToZip(zipWriter, file)
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	}

	return FileOperation{Success: true, Message: "ZIP归档创建成功"}
}

func createTarArchive(files []string, output string) FileOperation {
	tarFile, err := os.Create(output)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer tarFile.Close()

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	for _, file := range files {
		err := addFileToTar(tarWriter, file)
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	}

	return FileOperation{Success: true, Message: "TAR归档创建成功"}
}

func createTarGzArchive(files []string, output string) FileOperation {
	tarGzFile, err := os.Create(output)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer tarGzFile.Close()

	gzWriter := gzip.NewWriter(tarGzFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for _, file := range files {
		err := addFileToTar(tarWriter, file)
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	}

	return FileOperation{Success: true, Message: "TAR.GZ归档创建成功"}
}

func extractZipArchive(archive, output string) FileOperation {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer reader.Close()

	for _, file := range reader.File {
		err := extractFileFromZip(file, output)
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	}

	return FileOperation{Success: true, Message: "ZIP解压成功"}
}

func extractTarArchive(archive, output string) FileOperation {
	file, err := os.Open(archive)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer file.Close()

	reader := tar.NewReader(file)
	return extractFromTar(reader, output)
}

func extractTarGzArchive(archive, output string) FileOperation {
	file, err := os.Open(archive)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer gzReader.Close()

	reader := tar.NewReader(gzReader)
	return extractFromTar(reader, output)
}

func addFileToZip(zipWriter *zip.Writer, file string) error {
	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filepath.Base(file)
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		file, err := os.Open(file)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func addFileToTar(tarWriter *tar.Writer, file string) error {
	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	header.Name = filepath.Base(file)
	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		file, err := os.Open(file)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractFileFromZip(file *zip.File, output string) error {
	path := filepath.Join(output, file.Name)

	if file.FileInfo().IsDir() {
		return os.MkdirAll(path, file.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer writer.Close()

	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(writer, reader)
	return err
}

func extractFromTar(reader *tar.Reader, output string) FileOperation {
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}

		path := filepath.Join(output, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, header.FileInfo().Mode()); err != nil {
				return FileOperation{Success: false, Error: err}
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return FileOperation{Success: false, Error: err}
			}

			writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return FileOperation{Success: false, Error: err}
			}

			if _, err := io.Copy(writer, reader); err != nil {
				writer.Close()
				return FileOperation{Success: false, Error: err}
			}

			writer.Close()
		}
	}

	return FileOperation{Success: true, Message: "解压成功"}
}
