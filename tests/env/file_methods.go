/*
Copyright 2022 Citrix Systems, Inc
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package env

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func CreateAndWriteFile(path, contents string) error {
	// detect if file exists
	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if err != nil {
			return err
		}
		// Write to the file
		_, err = file.WriteString(contents)
		if err != nil {
			return err
		}
		// save changes
		err = file.Sync()
		if err != nil {
			return err
		}

		defer file.Close()
	}
	return nil
}

func DeleteFile(path string) error {
	// delete file
	var err = os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

func CopyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}

func SetCertEnv(certdir, rootcertpath, certchainpath, clientcertpath, keypath string) error {
	err := CopyFileContents(rootcertpath, certdir+"/root-cert.pem")
	if err != nil {
		return fmt.Errorf("Could not copy rootcert contents. Err=%s", err)
	}

	err = CopyFileContents(certchainpath, certdir+"/cert-chain.pem")
	if err != nil {
		return fmt.Errorf("Could not copy cert-chain contents. Err=%s", err)
	}

	err = CopyFileContents(clientcertpath, certdir+"/cert.pem")
	if err != nil {
		return fmt.Errorf("Could not copy client-cert contents. Err=%s", err)
	}

	err = CopyFileContents(keypath, certdir+"/key.pem")
	if err != nil {
		return fmt.Errorf("Could not copy key-file contents. Err=%s", err)
	}
	return nil
}

func getFileContent(fileName string) ([]byte, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Println("[ERROR] Reading File", data, err)
	}
	return data, err
}
func GetCertKeyData(certPath, keyPath string) ([]byte, []byte, error) {
	var certData, keyData []byte
	var err error
	certData, err = getFileContent(certPath)
	if err != nil {
		log.Println("[ERROR] Reading File:", certPath, err)
		return certData, keyData, err
	}
	if keyPath != "" {
		keyData, err = getFileContent(keyPath)
		if err != nil {
			log.Println("[ERROR] Reading File:", keyPath, err)
			return certData, keyData, err
		}
	}
	return certData, keyData, nil
}
