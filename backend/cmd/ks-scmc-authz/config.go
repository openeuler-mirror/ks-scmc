package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"sync"
)

var globalConfig *Config

type Config struct {
	filePath string
	fileLock sync.Mutex
	lock     sync.RWMutex

	SensitiveContainers map[string]interface{}
}

func initConfig(path string) error {
	globalConfig = &Config{
		filePath:            path,
		SensitiveContainers: make(map[string]interface{}),
	}

	return globalConfig.loadFile()
}

func (c *Config) loadFile() error {
	c.fileLock.Lock()
	defer c.fileLock.Unlock()

	// TODO ignore file not exist
	content, err := ioutil.ReadFile(c.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	if len(content) == 0 {
		return nil
	}

	if err := json.Unmarshal(content, c); err != nil {
		return err
	}
	return nil
}

func (c *Config) saveFile() error {
	c.fileLock.Lock()
	defer c.fileLock.Unlock()

	content, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// TODO create file if not exists
	if err := ioutil.WriteFile(c.filePath, content, 0644); err != nil {
		return nil
	}
	return nil
}

func (c *Config) addSensitiveContainers(ids []string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, id := range ids {
		c.SensitiveContainers[id] = 1
	}

	c.saveFile()
	return nil
}

func (c *Config) delSensitiveContainers(ids []string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, id := range ids {
		delete(c.SensitiveContainers, id)
	}

	c.saveFile()
	return nil
}

func (c *Config) isSensitiveContainers(id string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.SensitiveContainers[id]
	return ok
}
