package files

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"syscall"

	log "github.com/Sirupsen/logrus"

	"strconv"

	"github.com/labstack/echo"
)

type hash map[string]interface{}

type file_t struct {
	Id         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
}

func loadFileId(filepath string) (string, error) {
	pathp, err := syscall.UTF16PtrFromString(filepath)
	if err != nil {
		return "", err
	}
	h, err := syscall.CreateFile(pathp, 0, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(h)
	var i syscall.ByHandleFileInformation
	err = syscall.GetFileInformationByHandle(syscall.Handle(h), &i)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x", i.VolumeSerialNumber, i.FileIndexHigh, i.FileIndexLow), nil
}

/*
func Post(c *echo.Context) error {
	mr, err := c.Request().MultipartReader()
	if err != nil {
		log.Println(err)
		return err
	}
	for {
		p, err := mr.NextPart()
		if err != nil {
			log.Println(err)
			return err
		}
		if err == io.EOF {
			return err
		}
		if err != nil {
			log.Fatal(err)
		}
		var outfile *os.File
		if outfile, err = os.Create("C:/Users/Administrator/Desktop/" + p.FileName()); nil != err {
			return err
		}
		if _, err = io.Copy(outfile, p); nil != err {
			return err
		}
	}
	return nil
}*/

var kUploadDir string

// Get checks a chunk.
// If it doesn't exist then flowjs tries to upload it via Post.
func GetUpload(w http.ResponseWriter, r *http.Request) {
	sam := r.URL.Query()["sam"][0]

	log.Error(sam)
	kUploadDir = filepath.Join("C:/Users", sam, "Desktop/Nanocloud")
	if _, err := os.Stat(kUploadDir); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(kUploadDir, 0711)
			if err != nil {
				log.Error(err)
				http.Error(w, "Unable to create upload directory", http.StatusInternalServerError)
			}
		}
	}
	chunkPath := filepath.Join(
		kUploadDir,
		sam,
		"incomplete",
		r.FormValue("flowFilename"),
		r.FormValue("flowChunkNumber"),
	)
	if _, err := os.Stat(chunkPath); err != nil {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("chunk not found"))
		return
	}
}

// Post tries to get and save a chunk.
func Post(w http.ResponseWriter, r *http.Request) {
	sam := r.URL.Query()["sam"][0]

	log.Error(sam)
	kUploadDir = filepath.Join("C:/Users", sam, "Desktop/Nanocloud")
	if _, err := os.Stat(kUploadDir); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(kUploadDir, 0711)
			if err != nil {
				log.Error(err)
				http.Error(w, "Unable to create upload directory", http.StatusInternalServerError)
			}
		}
	}
	// get the multipart data
	err := r.ParseMultipartForm(2 * 1024 * 1024) // chunkSize
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chunkNum, err := strconv.Atoi(r.FormValue("flowChunkNumber"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalChunks, err := strconv.Atoi(r.FormValue("flowTotalChunks"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := r.FormValue("flowFilename")
	// module := r.FormValue("module")

	err = writeChunk(filepath.Join(kUploadDir, "incomplete", filename), strconv.Itoa(chunkNum), r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// it's done if it's not the last chunk
	if chunkNum < totalChunks {
		return
	}

	upPath := filepath.Join(kUploadDir, filename)

	// now finish the job
	err = assembleUpload(kUploadDir, filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to assemble the uploaded chunks")
		return
	}
	log.WithFields(log.Fields{
		"path": upPath,
	}).Info("file uploaded")
}

func writeChunk(path, chunkNum string, r *http.Request) error {
	// prepare the chunk folder
	err := os.MkdirAll(path, 02750)
	if err != nil {
		return err
	}
	// write the chunk
	fileIn, _, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer fileIn.Close()
	fileOut, err := os.Create(filepath.Join(path, chunkNum))
	if err != nil {
		return err
	}
	defer fileOut.Close()
	_, err = io.Copy(fileOut, fileIn)
	return err
}

func assembleUpload(path, filename string) error {

	// create final file to write to
	dst, err := os.Create(filepath.Join(path, filename))
	if err != nil {
		return err
	}
	defer dst.Close()

	chunkDirPath := filepath.Join(path, "incomplete", filename)
	fileInfos, err := ioutil.ReadDir(chunkDirPath)
	if err != nil {
		return err
	}
	sort.Sort(byChunk(fileInfos))
	for _, fs := range fileInfos {
		src, err := os.Open(filepath.Join(chunkDirPath, fs.Name()))
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		src.Close()
		if err != nil {
			return err
		}
	}
	os.RemoveAll(chunkDirPath)

	return nil
}

type byChunk []os.FileInfo

func (a byChunk) Len() int      { return len(a) }
func (a byChunk) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byChunk) Less(i, j int) bool {
	ai, _ := strconv.Atoi(a[i].Name())
	aj, _ := strconv.Atoi(a[j].Name())
	return ai < aj
}

func Get(c *echo.Context) error {
	filepath := c.Query("path")
	showHidden := c.Query("show_hidden") == "true"
	create := c.Query("create") == "true"

	if len(filepath) < 1 {
		return c.JSON(
			http.StatusBadRequest,
			hash{
				"error": "Path not specified",
			},
		)
	}

	s, err := os.Stat(filepath)
	if err != nil {
		fmt.Println(err.(*os.PathError).Err.Error())
		m := err.(*os.PathError).Err.Error()
		if m == "no such file or directory" || m == "The system cannot find the file specified." {
			if create {
				err := os.MkdirAll(filepath, 0777)
				if err != nil {
					return err
				}
				s, err = os.Stat(filepath)
				if err != nil {
					return err
				}
			} else {
				return c.JSON(
					http.StatusNotFound,
					hash{
						"error": "no such file or directory",
					},
				)
			}
		} else {
			return err
		}
	}

	if s.Mode().IsDir() {
		f, err := os.Open(filepath)
		if err != nil {
			return err
		}
		defer f.Close()

		files, err := f.Readdir(-1)
		if err != nil {
			return err
		}

		rt := make([]file_t, 0)

		for _, file := range files {
			name := file.Name()
			if !showHidden {
				sys := file.Sys().(*syscall.Win32FileAttributeData)

				if sys.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN == syscall.FILE_ATTRIBUTE_HIDDEN {
					continue
				}

			}

			fullpath := path.Join(filepath, name)
			id, err := loadFileId(fullpath)
			if err != nil {
				log.Errorf("Cannot retrieve file id for file: %s: %s", fullpath, err.Error())
				continue
			}

			f := file_t{
				Id:   id,
				Type: "file",
			}

			attr := make(map[string]interface{}, 0)
			f.Attributes = attr

			attr["mod_time"] = file.ModTime().Unix()
			attr["size"] = file.Size()
			attr["name"] = name

			if file.IsDir() {
				attr["type"] = "directory"
			} else {
				attr["type"] = "regular file"
			}
			rt = append(rt, f)
		}
		/*
		 * The Content-Length is not set is the buffer length is more than 2048
		 */
		b, err := json.Marshal(hash{
			"data": rt,
		})
		if err != nil {
			return err
		}

		r := c.Response()
		r.Header().Set("Content-Length", strconv.Itoa(len(b)))
		r.Header().Set("Content-Type", "application/json; charset=utf-8")
		r.Write(b)
		return nil
	}

	return c.File(
		filepath,
		s.Name(),
		true,
	)
}
