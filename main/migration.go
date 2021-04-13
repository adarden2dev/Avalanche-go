package main

import (
	"fmt"

	"github.com/ava-labs/avalanchego/config"
	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/node"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/spf13/viper"
)

type migrationManager struct {
	binaryManager *binaryManager
	nodeConfig    node.Config
	log           logging.Logger
	viperConfig   *viper.Viper
}

func newMigrationManager(binaryManager *binaryManager, v *viper.Viper, nConfig node.Config, log logging.Logger) *migrationManager {
	return &migrationManager{
		binaryManager: binaryManager,
		nodeConfig:    nConfig,
		log:           log,
		viperConfig:   v,
	}
}

func (m *migrationManager) migrate() error {
	shouldMigrate, err := m.shouldMigrate()
	if err != nil {
		return err
	}
	if !shouldMigrate {
		return nil
	}
	return m.runMigration()
}

func (m *migrationManager) shouldMigrate() (bool, error) {
	if !m.nodeConfig.DBEnabled {
		return false, nil
	}
	dbManager, err := manager.New(m.nodeConfig.DBPath, logging.NoLog{}, config.DBVersion, true)
	if err != nil {
		return false, fmt.Errorf("couldn't create db manager at %s: %w", m.nodeConfig.DBPath, err)
	}
	defer func() {
		if err := dbManager.Shutdown(); err != nil {
			m.log.Error("error shutting down db manager: %s", err)
		}
	}()
	currentDBBootstrapped, err := dbManager.CurrentDBBootstrapped()
	if err != nil {
		return false, fmt.Errorf("couldn't get if database version %s is bootstrapped: %w", config.DBVersion, err)
	}
	if currentDBBootstrapped {
		return false, nil
	}
	_, exists := dbManager.Previous()
	return exists, nil
}

// Run two nodes at once: one is a version before the database upgrade and the other after.
// The latter will bootstrap from the former. Its staking port and HTTP port are 2
// greater than the staking/HTTP ports in [v].
// When the new node version is done bootstrapping, both nodes are stopped.
// Returns nil if the new node version successfully bootstrapped.
func (m *migrationManager) runMigration() error {
	prevVersionNode, err := m.binaryManager.runPreviousVersion(node.PreviousVersion.AsVersion(), m.viperConfig)
	if err != nil {
		return fmt.Errorf("couldn't start old version during migration: %w", err)
	}
	defer func() {
		if err := m.binaryManager.kill(prevVersionNode.processID); err != nil {
			m.log.Error("error while killing previous version: %s", err)
		}
	}()

	currentVersionNode, err := m.binaryManager.runCurrentVersion(m.viperConfig, true, m.nodeConfig.NodeID)
	if err != nil {
		return fmt.Errorf("couldn't start current version during migration: %w", err)
	}
	defer func() {
		if err := m.binaryManager.kill(currentVersionNode.processID); err != nil {
			m.log.Error("error while killing current version: %s", err)
		}
	}()

	for {
		select {
		case err := <-prevVersionNode.errChan:
			return fmt.Errorf("previous version stopped with: %s", err)
		case <-currentVersionNode.errChan:
			if currentVersionNode.exitCode != constants.ExitCodeDoneMigrating {
				return fmt.Errorf("current version died with exit code %d", currentVersionNode.exitCode)
			}
			return nil
		}
	}
}
