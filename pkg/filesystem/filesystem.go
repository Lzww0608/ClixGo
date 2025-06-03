/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-03 19:25:00
* @Description: 文件系统操作的核心实现，提供文件管理、权限控制等功能
 */

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

// FileInfo 文件信息结构体
// 包含文件的基本属性、权限、校验和等详细信息
type FileInfo struct {
	Name        string            // 文件名
	Path        string            // 文件完整路径（符号链接会显示目标）
	Size        int64             // 文件大小（字节）
	Mode        fs.FileMode       // 文件模式和权限
	ModTime     time.Time         // 最后修改时间
	IsDir       bool              // 是否为目录
	IsSymlink   bool              // 是否为符号链接
	Owner       string            // 文件所有者用户ID
	Group       string            // 文件所属组ID
	Checksum    map[string]string // 文件校验和（MD5、SHA1、SHA256）
	Permissions string            // 权限字符串表示
	ContentType string            // 文件MIME类型
}

// FileOperation 文件操作结果
// 封装操作成功状态、消息和错误信息
type FileOperation struct {
	Success bool   // 操作是否成功
	Message string // 操作结果消息
	Error   error  // 错误信息（如果有）
}

// ListFiles 列出指定目录的文件和子目录
//
// 参数:
//   - path: 目录路径
//   - recursive: 是否递归列出子目录内容
//   - showHidden: 是否显示隐藏文件（以.开头的文件）
//
// 返回:
//   - []FileInfo: 文件信息列表
//   - error: 列举过程中的错误，nil表示成功
//
// 该函数会收集每个文件的详细信息，包括校验和、权限、MIME类型等
func ListFiles(path string, recursive bool, showHidden bool) ([]FileInfo, error) {
	cleanedPath := filepath.Clean(path)
	var fileInfoList []FileInfo

	// 确保路径存在
	_, err := os.Stat(cleanedPath)
	if err != nil {
		return nil, err
	}

	walkFunction := func(currentFilePath string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏文件（如果需要）
		if !showHidden && strings.HasPrefix(fileInfo.Name(), ".") {
			if fileInfo.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 如果不是递归模式，只处理当前目录下的文件和目录
		if !recursive && filepath.Dir(currentFilePath) != cleanedPath {
			// 如果是子目录，跳过其内容
			if fileInfo.IsDir() && currentFilePath != cleanedPath {
				return filepath.SkipDir
			}
			// 如果不是当前目录下的文件，跳过
			if filepath.Dir(currentFilePath) != cleanedPath {
				return nil
			}
		}

		detailedFileInfo := FileInfo{
			Name:    fileInfo.Name(),
			Path:    currentFilePath,
			Size:    fileInfo.Size(),
			Mode:    fileInfo.Mode(),
			ModTime: fileInfo.ModTime(),
			IsDir:   fileInfo.IsDir(),
		}

		// 获取符号链接信息
		if fileInfo.Mode()&fs.ModeSymlink != 0 {
			detailedFileInfo.IsSymlink = true
			if linkTarget, err := os.Readlink(currentFilePath); err == nil {
				detailedFileInfo.Path = fmt.Sprintf("%s -> %s", currentFilePath, linkTarget)
			}
		}

		// 获取文件所有者信息
		if statInfo, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
			detailedFileInfo.Owner = fmt.Sprintf("%d", statInfo.Uid)
			detailedFileInfo.Group = fmt.Sprintf("%d", statInfo.Gid)
		}

		// 计算文件校验和
		if !fileInfo.IsDir() {
			detailedFileInfo.Checksum = calculateChecksums(currentFilePath)
		}

		// 获取文件权限
		detailedFileInfo.Permissions = fileInfo.Mode().String()

		// 获取文件类型
		if !fileInfo.IsDir() {
			detailedFileInfo.ContentType = detectContentType(currentFilePath)
		}

		fileInfoList = append(fileInfoList, detailedFileInfo)
		return nil
	}

	if err := filepath.Walk(cleanedPath, walkFunction); err != nil {
		return nil, err
	}

	return fileInfoList, nil
}

// CopyFile 复制文件或目录
//
// 参数:
//   - src: 源文件或目录路径
//   - dst: 目标文件或目录路径
//
// 返回:
//   - FileOperation: 操作结果，包含成功状态和错误信息
//
// 如果源是目录，会递归复制整个目录树
func CopyFile(src, dst string) FileOperation {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	if sourceInfo.IsDir() {
		return copyDirectory(src, dst)
	}

	return copySingleFile(src, dst)
}

// MoveFile 移动或重命名文件或目录
//
// 参数:
//   - src: 源文件或目录路径
//   - dst: 目标文件或目录路径
//
// 返回:
//   - FileOperation: 操作结果，包含成功状态和错误信息
func MoveFile(src, dst string) FileOperation {
	err := os.Rename(src, dst)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	return FileOperation{Success: true, Message: "文件移动成功"}
}

// DeleteFile 删除文件或目录
//
// 参数:
//   - path: 要删除的文件或目录路径
//   - recursive: 是否递归删除目录内容
//
// 返回:
//   - FileOperation: 操作结果，包含成功状态和错误信息
//
// 对于目录，如果recursive为false且目录非空，操作会失败
func DeleteFile(path string, recursive bool) FileOperation {
	targetInfo, err := os.Stat(path)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	if targetInfo.IsDir() && !recursive {
		return FileOperation{Success: false, Error: fmt.Errorf("目录非空，请使用递归删除")}
	}

	if targetInfo.IsDir() {
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
//
// 参数:
//   - path: 要创建的目录路径
//   - parents: 是否创建父目录（类似mkdir -p）
//
// 返回:
//   - FileOperation: 操作结果，包含成功状态和错误信息
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

// ChangePermissions 修改文件或目录权限
//
// 参数:
//   - path: 文件或目录路径
//   - mode: 新的权限模式
//   - recursive: 是否递归修改目录内容权限
//
// 返回:
//   - FileOperation: 操作结果，包含成功状态和错误信息
func ChangePermissions(path string, mode os.FileMode, recursive bool) FileOperation {
	if recursive {
		err := filepath.Walk(path, func(currentPath string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			return os.Chmod(currentPath, mode)
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

	// 读取更多内容以便更准确地检测类型
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "unknown"
	}

	// 如果读取的内容少于512字节，调整buffer大小
	if n < 512 {
		buffer = buffer[:n]
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
	// 先将路径标准化
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	// 检查目标路径是否是源路径的子目录，避免无限递归
	if strings.HasPrefix(dst, src) {
		return FileOperation{Success: false, Error: fmt.Errorf("目标目录不能是源目录的子目录")}
	}

	// 确保目标目录存在
	if err := os.MkdirAll(dst, 0755); err != nil {
		return FileOperation{Success: false, Error: err}
	}

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

	if info.IsDir() && !strings.HasSuffix(header.Name, "/") {
		header.Name += "/"
	}

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
	// 检查路径是否尝试遍历到输出目录之外
	destPath := filepath.Join(output, file.Name)

	// 安全检查：确保最终路径在输出目录内
	if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(output)) {
		return fmt.Errorf("非法的文件路径: %s (路径遍历尝试)", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(destPath, file.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	writer, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
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

		destPath := filepath.Join(output, header.Name)

		// 安全检查：确保最终路径在输出目录内
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(output)) {
			return FileOperation{Success: false, Error: fmt.Errorf("非法的文件路径: %s (路径遍历尝试)", header.Name)}
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, header.FileInfo().Mode()); err != nil {
				return FileOperation{Success: false, Error: err}
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return FileOperation{Success: false, Error: err}
			}

			writer, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode())
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
