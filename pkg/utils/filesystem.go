/*
* @Author: Lzww0608
* @Date: 2025-06-14 11:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-14 11:00:00
* @Description: 文件系统操作功能 (从pkg/filesystem迁移到utils)
 */

package utils

import (
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
func MoveFile(src, dst string) FileOperation {
	err := os.Rename(src, dst)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	return FileOperation{Success: true, Message: "文件移动成功"}
}

// DeleteFile 删除文件或目录
func DeleteFile(path string, recursive bool) FileOperation {
	targetInfo, err := os.Stat(path)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	if targetInfo.IsDir() {
		if recursive {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
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

// ChangePermissions 修改文件或目录权限
func ChangePermissions(path string, mode os.FileMode, recursive bool) FileOperation {
	if recursive {
		err := filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			return os.Chmod(walkPath, mode)
		})
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	} else {
		if err := os.Chmod(path, mode); err != nil {
			return FileOperation{Success: false, Error: err}
		}
	}

	return FileOperation{Success: true, Message: "权限修改成功"}
}

// ChangeOwner 修改文件或目录所有者
func ChangeOwner(path string, uid, gid int, recursive bool) FileOperation {
	if recursive {
		err := filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			return os.Chown(walkPath, uid, gid)
		})
		if err != nil {
			return FileOperation{Success: false, Error: err}
		}
	} else {
		if err := os.Chown(path, uid, gid); err != nil {
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

// 内部辅助函数
func calculateChecksums(path string) map[string]string {
	checksums := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		return checksums
	}
	defer file.Close()

	// MD5
	md5Hash := md5.New()
	sha1Hash := sha1.New()
	sha256Hash := sha256.New()

	multiWriter := io.MultiWriter(md5Hash, sha1Hash, sha256Hash)

	if _, err := io.Copy(multiWriter, file); err != nil {
		return checksums
	}

	checksums["md5"] = hex.EncodeToString(md5Hash.Sum(nil))
	checksums["sha1"] = hex.EncodeToString(sha1Hash.Sum(nil))
	checksums["sha256"] = hex.EncodeToString(sha256Hash.Sum(nil))

	return checksums
}

func detectContentType(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "application/octet-stream"
	}

	return http.DetectContentType(buffer[:n])
}

func copySingleFile(src, dst string) FileOperation {
	sourceFile, err := os.Open(src)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	// 复制文件权限
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	err = os.Chmod(dst, sourceInfo.Mode())
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	return FileOperation{Success: true, Message: "文件复制成功"}
}

func copyDirectory(src, dst string) FileOperation {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	err = os.MkdirAll(dst, sourceInfo.Mode())
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	directory, err := os.Open(src)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}
	defer directory.Close()

	objects, err := directory.Readdir(-1)
	if err != nil {
		return FileOperation{Success: false, Error: err}
	}

	for _, obj := range objects {
		srcPath := filepath.Join(src, obj.Name())
		dstPath := filepath.Join(dst, obj.Name())

		if obj.IsDir() {
			result := copyDirectory(srcPath, dstPath)
			if !result.Success {
				return result
			}
		} else {
			result := copySingleFile(srcPath, dstPath)
			if !result.Success {
				return result
			}
		}
	}

	return FileOperation{Success: true, Message: "目录复制成功"}
}
