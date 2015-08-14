package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	sl "github.com/maximilien/softlayer-go/softlayer"

	bslcommon "github.com/maximilien/bosh-softlayer-cpi/softlayer/common"
	bslcdisk "github.com/maximilien/bosh-softlayer-cpi/softlayer/disk"
	bslcstem "github.com/maximilien/bosh-softlayer-cpi/softlayer/stemcell"
	bslcvmpool "github.com/maximilien/bosh-softlayer-cpi/softlayer/vm/pool"

	common "github.com/maximilien/bosh-softlayer-cpi/common"
	util "github.com/maximilien/bosh-softlayer-cpi/util"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	sldatatypes "github.com/maximilien/softlayer-go/data_types"
)

const (
	softLayerVMtag   = "SoftLayerVM"
	ROOT_USER_NAME   = "root"
	deleteVMLogTag   = "DeleteVM"
	osReloadVMLogTag = "OSReload"

	TIMEOUT_TRANSACTIONS_DELETE_VM   = 60 * time.Minute
	TIMEOUT_TRANSACTIONS_CREATE_VM   = 100 * time.Minute
	TIMEOUT_TRANSACTIONS_OSRELOAD_VM = 24 * time.Hour
)

type SoftLayerVM struct {
	id int

	softLayerClient sl.Client
	agentEnvService AgentEnvService

	sshClient util.SshClient

	logger boshlog.Logger

	timeoutForActiveTransactions time.Duration
}

func NewSoftLayerVM(id int, softLayerClient sl.Client, sshClient util.SshClient, agentEnvService AgentEnvService, logger boshlog.Logger, timeoutForActiveTransactions time.Duration) SoftLayerVM {
	bslcommon.TIMEOUT = 10 * time.Minute
	bslcommon.POLLING_INTERVAL = 10 * time.Second

	return SoftLayerVM{
		id: id,

		softLayerClient: softLayerClient,
		agentEnvService: agentEnvService,

		sshClient: sshClient,

		logger: logger,

		timeoutForActiveTransactions: timeoutForActiveTransactions,
	}
}

func (vm SoftLayerVM) ID() int { return vm.id }

func (vm SoftLayerVM) Delete() error {

	if strings.ToUpper(common.GetOSEnvVariable("OS_RELOAD_ENABLED", "TRUE")) == "FALSE" {
		return vm.DeleteVM()
	}

	err := bslcvmpool.InitVMPoolDB()
	if err != nil {
		return bosherr.WrapError(err, "Failed to initialize VM pool DB")
	}

	vmInfoDB := bslcvmpool.NewVMInfoDB(vm.id, "", "", "", "", vm.logger)
	defer vmInfoDB.CloseDB()

	err = vmInfoDB.QueryVMInfobyID()
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("Failed to query VM info by given ID %d", vm.id))
	}
	vm.logger.Info(softLayerCreatorLogTag, fmt.Sprintf("vmInfoDB.vmProperties.id is %d", vmInfoDB.VmProperties.Id))

	if vmInfoDB.VmProperties.Id != 0 {
		vm.logger.Info(softLayerCreatorLogTag, fmt.Sprintf("Release the VM with id %d back to the VM pool", vmInfoDB.VmProperties.Id))
		vmInfoDB.VmProperties.InUse = "f"
		err = vmInfoDB.UpdateVMInfoByID()
		if err != nil {
			return bosherr.WrapError(err, fmt.Sprintf("Failed to query VM info by given ID %d", vm.id))
		} else {
			return nil
		}
	}

	return vm.DeleteVM()
}

func (vm SoftLayerVM) DeleteVM() error {
	virtualGuestService, err := vm.softLayerClient.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return bosherr.WrapError(err, "Creating SoftLayer VirtualGuestService from client")
	}

	vmCID := vm.ID()
	err = bslcommon.WaitForVirtualGuestToHaveNoRunningTransactions(vm.softLayerClient, vmCID, vm.timeoutForActiveTransactions, bslcommon.POLLING_INTERVAL)
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("Waiting for VirtualGuest `%d` to have no pending transactions before deleting vm", vmCID))
	}

	deleted, err := virtualGuestService.DeleteObject(vm.ID())
	if err != nil {
		return bosherr.WrapError(err, "Deleting SoftLayer VirtualGuest from client")
	}

	if !deleted {
		return bosherr.WrapError(nil, "Did not delete SoftLayer VirtualGuest from client")
	}

	err = vm.postCheckActiveTransactionsForDeleteVM(vm.softLayerClient, vmCID, vm.timeoutForActiveTransactions, bslcommon.POLLING_INTERVAL)
	if err != nil {
		return err
	}

	return nil
}

func (vm SoftLayerVM) Reboot() error {
	virtualGuestService, err := vm.softLayerClient.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return bosherr.WrapError(err, "Creating SoftLayer VirtualGuestService from client")
	}

	rebooted, err := virtualGuestService.RebootSoft(vm.ID())
	if err != nil {
		return bosherr.WrapError(err, "Rebooting (soft) SoftLayer VirtualGuest from client")
	}

	if !rebooted {
		return bosherr.WrapError(nil, "Did not reboot (soft) SoftLayer VirtualGuest from client")
	}

	return nil
}

func (vm SoftLayerVM) ReloadOS(stemcell bslcstem.Stemcell) error {
	reload_OS_Config := sldatatypes.Image_Template_Config{
		ImageTemplateId: strconv.Itoa(stemcell.ID()),
	}

	virtualGuestService, err := vm.softLayerClient.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return bosherr.WrapError(err, "Creating VirtualGuestService from SoftLayer client")
	}

	err = bslcommon.WaitForVirtualGuestToHaveNoRunningTransactions(vm.softLayerClient, vm.ID(), vm.timeoutForActiveTransactions, bslcommon.POLLING_INTERVAL)
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("Waiting for VirtualGuest %d to have no pending transactions before os reload", vm.ID()))
	}
	vm.logger.Info(osReloadVMLogTag, fmt.Sprintf("No transaction is running on this VM %d", vm.ID()))

	err = virtualGuestService.ReloadOperatingSystem(vm.ID(), reload_OS_Config)
	if err != nil {
		return bosherr.WrapError(err, "Failed to reload OS on the specified VirtualGuest from SoftLayer client")
	}

	time.Sleep(time.Second * 30)

	err = vm.postCheckActiveTransactionsForOSReload(vm.softLayerClient, vm.timeoutForActiveTransactions, bslcommon.POLLING_INTERVAL)
	if err != nil {
		return err
	}

	return nil
}

func (vm SoftLayerVM) SetMetadata(vmMetadata VMMetadata) error {
	tags, err := vm.extractTagsFromVMMetadata(vmMetadata)
	if err != nil {
		return err
	}

	//Check below needed since Golang strings.Split return [""] on strings.Split("", ",")
	if len(tags) == 1 && tags[0] == "" {
		return nil
	}

	if len(tags) == 0 {
		return nil
	}

	virtualGuestService, err := vm.softLayerClient.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return bosherr.WrapError(err, "Creating SoftLayer VirtualGuestService from client")
	}

	success, err := virtualGuestService.SetTags(vm.ID(), tags)
	if !success {
		return bosherr.WrapErrorf(err, "Settings tags on SoftLayer VirtualGuest `%d`", vm.ID())
	}

	if err != nil {
		return bosherr.WrapErrorf(err, "Settings tags on SoftLayer VirtualGuest `%d`", vm.ID())
	}

	return nil
}

func (vm SoftLayerVM) ConfigureNetworks(networks Networks) error {
	return NotSupportedError{}
}

func (vm SoftLayerVM) AttachDisk(disk bslcdisk.Disk) error {
	virtualGuest, volume, err := vm.fetchVMandIscsiVolume(vm.ID(), disk.ID())
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("Failed to fetch disk `%d` and virtual gusest `%d`", disk.ID(), virtualGuest.Id))
	}

	networkStorageService, err := vm.softLayerClient.GetSoftLayer_Network_Storage_Service()
	if err != nil {
		return bosherr.WrapError(err, "Can not get network storage service.")
	}

	allowed, err := networkStorageService.HasAllowedVirtualGuest(disk.ID(), vm.ID())
	if err == nil && allowed == false {
		err = networkStorageService.AttachIscsiVolume(virtualGuest, disk.ID())
	}
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("Failed to grant access of disk `%d` from virtual gusest `%d`", disk.ID(), virtualGuest.Id))
	}

	_, err = vm.attachVolumeBasedOnShellScript(virtualGuest, volume)
	if err != nil {
		return bosherr.WrapErrorf(err, "Failed to attach volume with id %d to virtual guest with id: %d.", volume.Id, virtualGuest.Id)
	}

	metadata, err := bslcommon.GetUserMetadataOnVirtualGuest(vm.softLayerClient, virtualGuest.Id)
	if err != nil {
		return bosherr.WrapErrorf(err, "Failed to get metadata from virtual guest with id: %d.", virtualGuest.Id)
	}

	old_agentEnv, err := NewAgentEnvFromJSON(metadata)
	if err != nil {
		return bosherr.WrapErrorf(err, "Failed to unmarshal metadata from virutal guest with id: %d.", virtualGuest.Id)
	}
	new_agentEnv := old_agentEnv.AttachPersistentDisk(strconv.Itoa(disk.ID()), "/dev/sda") // Naming convention for iscsi driver in Softlayer.

	metadata, err = json.Marshal(new_agentEnv)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling agent environment metadata")
	}

	err = bslcommon.ConfigureMetadataOnVirtualGuest(vm.softLayerClient, virtualGuest.Id, string(metadata), bslcommon.TIMEOUT, bslcommon.POLLING_INTERVAL)
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("Configuring metadata on VirtualGuest `%d`", virtualGuest.Id))
	}

	return nil
}

func (vm SoftLayerVM) DetachDisk(disk bslcdisk.Disk) error {
	virtualGuest, volume, err := vm.fetchVMandIscsiVolume(vm.ID(), disk.ID())
	if err != nil {
		return err
	}

	networkStorageService, err := vm.softLayerClient.GetSoftLayer_Network_Storage_Service()
	if err != nil {
		return bosherr.WrapError(err, "Can not get network storage service.")
	}

	allowed, err := networkStorageService.HasAllowedVirtualGuest(disk.ID(), vm.ID())
	if err == nil && allowed == true {
		err = networkStorageService.DetachIscsiVolume(virtualGuest, disk.ID())
	}
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("Failed to revoke access of disk `%d` from virtual gusest `%d`", disk.ID(), virtualGuest.Id))
	}

	err = vm.detachVolumeBasedOnShellScript(virtualGuest, volume)
	if err != nil {
		return bosherr.WrapErrorf(err, "Failed to detach volume with id %d from virtual guest with id: %d.", volume.Id, virtualGuest.Id)
	}

	metadata, err := bslcommon.GetUserMetadataOnVirtualGuest(vm.softLayerClient, virtualGuest.Id)
	if err != nil {
		return bosherr.WrapErrorf(err, "Failed to get metadata from virtual guest with id: %d.", virtualGuest.Id)
	}

	old_agentEnv, err := NewAgentEnvFromJSON(metadata)
	new_agentEnv := old_agentEnv.DetachPersistentDisk(strconv.Itoa(disk.ID()))

	metadata, err = json.Marshal(new_agentEnv)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling agent environment metadata")
	}

	err = bslcommon.ConfigureMetadataOnVirtualGuest(vm.softLayerClient, virtualGuest.Id, string(metadata), bslcommon.TIMEOUT, bslcommon.POLLING_INTERVAL)
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("Configuring metadata on VirtualGuest `%d`", virtualGuest.Id))
	}

	return nil
}

// Private methods
func (vm SoftLayerVM) extractTagsFromVMMetadata(vmMetadata VMMetadata) ([]string, error) {
	tags := []string{}
	for key, value := range vmMetadata {
		if key == "tags" {
			stringValue, ok := value.(string)
			if !ok {
				return []string{}, bosherr.Errorf("Could not convert tags metadata value `%v` to string", value)
			}

			tags = vm.parseTags(stringValue)
		}
	}

	return tags, nil
}

func (vm SoftLayerVM) parseTags(value string) []string {
	return strings.Split(value, ",")
}

func (vm SoftLayerVM) fetchVMandIscsiVolume(vmId int, volumeId int) (datatypes.SoftLayer_Virtual_Guest, datatypes.SoftLayer_Network_Storage, error) {
	virtualGuestService, err := vm.softLayerClient.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, datatypes.SoftLayer_Network_Storage{}, bosherr.WrapError(err, "Can not get softlayer virtual guest service.")
	}

	networkStorageService, err := vm.softLayerClient.GetSoftLayer_Network_Storage_Service()
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, datatypes.SoftLayer_Network_Storage{}, bosherr.WrapError(err, "Can not get network storage service.")
	}

	virtualGuest, err := virtualGuestService.GetObject(vmId)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, datatypes.SoftLayer_Network_Storage{}, bosherr.WrapErrorf(err, "Can not get virtual guest with id: %d", vmId)
	}

	volume, err := networkStorageService.GetIscsiVolume(volumeId)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, datatypes.SoftLayer_Network_Storage{}, bosherr.WrapErrorf(err, "Failed to get iSCSI volume with id: %d", volumeId)
	}

	return virtualGuest, volume, nil
}

func (vm SoftLayerVM) attachVolumeBasedOnShellScript(virtualGuest datatypes.SoftLayer_Virtual_Guest, volume datatypes.SoftLayer_Network_Storage) (string, error) {
	command := fmt.Sprintf(`
		export PATH=/etc/init.d:$PATH
		cp /etc/iscsi/iscsid.conf{,.save}
		sed '/^node.startup/s/^.*/node.startup = automatic/' -i /etc/iscsi/iscsid.conf
		sed '/^#node.session.auth.authmethod/s/#//' -i /etc/iscsi/iscsid.conf
		sed '/^#node.session.auth.username / {s/#//; s/ username/ %s/}' -i /etc/iscsi/iscsid.conf
		sed '/^#node.session.auth.password / {s/#//; s/ password/ %s/}' -i /etc/iscsi/iscsid.conf
		sed '/^#discovery.sendtargets.auth.username / {s/#//; s/ username/ %s/}' -i /etc/iscsi/iscsid.conf
		sed '/^#discovery.sendtargets.auth.password / {s/#//; s/ password/ %s/}' -i /etc/iscsi/iscsid.conf
		open-iscsi restart
		rm -r /etc/iscsi/send_targets
		open-iscsi stop
		open-iscsi start
		iscsiadm -m discovery -t sendtargets -p %s
		open-iscsi restart`,
		volume.Username,
		volume.Password,
		volume.Username,
		volume.Password,
		volume.ServiceResourceBackendIpAddress,
	)

	_, err := vm.sshClient.ExecCommand(ROOT_USER_NAME, vm.getRootPassword(virtualGuest), virtualGuest.PrimaryIpAddress, command)
	if err != nil {
		return "", err
	}

	_, deviceName, err := vm.findIscsiDeviceNameBasedOnShellScript(virtualGuest, volume)
	if err != nil {
		return "", err
	}

	return deviceName, nil
}

func (vm SoftLayerVM) detachVolumeBasedOnShellScript(virtualGuest datatypes.SoftLayer_Virtual_Guest, volume datatypes.SoftLayer_Network_Storage) error {
	targetName, _, err := vm.findIscsiDeviceNameBasedOnShellScript(virtualGuest, volume)
	command := fmt.Sprintf(`
		iscsiadm -m node -T %s -u
		iscsiadm -m node -o delete -T %s`,
		targetName,
		targetName,
	)

	_, err = vm.sshClient.ExecCommand(ROOT_USER_NAME, vm.getRootPassword(virtualGuest), virtualGuest.PrimaryIpAddress, command)

	return err
}

func (vm SoftLayerVM) findIscsiDeviceNameBasedOnShellScript(virtualGuest datatypes.SoftLayer_Virtual_Guest, volume datatypes.SoftLayer_Network_Storage) (targetName string, deviceName string, err error) {
	command := `
		sleep 1
		iscsiadm -m session -P3 | sed -n  "/Target:/s/Target: //p; /Attached scsi disk /{ s/Attached scsi disk //; s/State:.*//p}"`

	output, err := vm.sshClient.ExecCommand(ROOT_USER_NAME, vm.getRootPassword(virtualGuest), virtualGuest.PrimaryIpAddress, command)
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(strings.Trim(output, "\n"), "\n")

	for i := 0; i < len(lines); i += 2 {
		if strings.Contains(lines[i], strings.ToLower(volume.Username)) {
			return strings.Trim(lines[i], "\t"), strings.Trim(lines[i+1], "\t"), nil
		}
	}

	return "", "", errors.New(fmt.Sprintf("Can not find matched iSCSI device for user name: %s", volume.Username))
}

func (vm SoftLayerVM) getRootPassword(virtualGuest datatypes.SoftLayer_Virtual_Guest) string {
	passwords := virtualGuest.OperatingSystem.Passwords

	for _, password := range passwords {
		if password.Username == ROOT_USER_NAME {
			return password.Password
		}
	}

	return ""
}

func (vm SoftLayerVM) postCheckActiveTransactionsForOSReload(softLayerClient sl.Client, timeout, pollingInterval time.Duration) error {
	virtualGuestService, err := softLayerClient.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return bosherr.WrapError(err, "Creating VirtualGuestService from SoftLayer client")
	}

	totalTime := time.Duration(0)
	for totalTime < timeout {
		activeTransactions, err := virtualGuestService.GetActiveTransactions(vm.ID())
		if err != nil {
			return bosherr.WrapError(err, "Getting active transactions from SoftLayer client")
		}

		if len(activeTransactions) > 0 {
			vm.logger.Info(osReloadVMLogTag, "OS Reload transaction started")
			break
		}

		totalTime += pollingInterval
		time.Sleep(pollingInterval)
	}

	if totalTime >= timeout {
		return errors.New(fmt.Sprintf("Waiting for OS Reload transaction to start TIME OUT!"))
	}

	err = bslcommon.WaitForVirtualGuest(vm.softLayerClient, vm.ID(), "RUNNING", bslcommon.TIMEOUT, bslcommon.POLLING_INTERVAL)
	if err != nil {
		return bosherr.WrapError(err, fmt.Sprintf("PowerOn failed with VirtualGuest id %d", vm.ID()))
	}

	vm.logger.Info(osReloadVMLogTag, fmt.Sprintf("The virtual guest %d is powered on", vm.ID()))

	return nil
}

func (vm SoftLayerVM) postCheckActiveTransactionsForDeleteVM(softLayerClient sl.Client, virtualGuestId int, timeout, pollingInterval time.Duration) error {
	virtualGuestService, err := softLayerClient.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return bosherr.WrapError(err, "Creating VirtualGuestService from SoftLayer client")
	}

	totalTime := time.Duration(0)
	for totalTime < timeout {
		activeTransactions, err := virtualGuestService.GetActiveTransactions(virtualGuestId)
		if err != nil {
			return bosherr.WrapError(err, "Getting active transactions from SoftLayer client")
		}

		if len(activeTransactions) > 0 {
			vm.logger.Info(deleteVMLogTag, "Delete VM transaction started", nil)
			break
		}

		totalTime += pollingInterval
		time.Sleep(pollingInterval)
	}

	if totalTime >= timeout {
		return errors.New(fmt.Sprintf("Waiting for DeleteVM transaction to start TIME OUT!"))
	}

	totalTime = time.Duration(0)
	for totalTime < timeout {
		vm1, err := virtualGuestService.GetObject(virtualGuestId)
		if err != nil || vm1.Id == 0 {
			vm.logger.Info(deleteVMLogTag, "VM doesn't exist. Delete done", nil)
			break
		}

		activeTransaction, err := virtualGuestService.GetActiveTransaction(virtualGuestId)
		if err != nil {
			return bosherr.WrapError(err, "Getting active transactions from SoftLayer client")
		}

		averageTransactionDuration, err := strconv.ParseFloat(activeTransaction.TransactionStatus.AverageDuration, 32)
		if err != nil {
			return bosherr.WrapError(err, "Parsing float for average transaction duration")
		}

		if averageTransactionDuration > 30 {
			vm.logger.Info(deleteVMLogTag, "Deleting VM instance had been launched and it is a long transaction. Please check Softlayer Portal", nil)
			break
		}

		vm.logger.Info(deleteVMLogTag, "This is a short transaction, waiting for all active transactions to complete", nil)
		totalTime += pollingInterval
		time.Sleep(pollingInterval)
	}

	if totalTime >= timeout {
		return errors.New(fmt.Sprintf("After deleting a vm, waiting for active transactions to complete TIME OUT!"))
	}

	return nil
}

func (vm SoftLayerVM) execCommand(virtualGuest datatypes.SoftLayer_Virtual_Guest, command string) (string, error) {
	result, err := vm.sshClient.ExecCommand(ROOT_USER_NAME, vm.getRootPassword(virtualGuest), virtualGuest.PrimaryIpAddress, command)
	return result, err
}

func (vm SoftLayerVM) uploadFile(virtualGuest datatypes.SoftLayer_Virtual_Guest, srcFile string, destFile string) error {
	err := vm.sshClient.UploadFile(ROOT_USER_NAME, vm.getRootPassword(virtualGuest), virtualGuest.PrimaryIpAddress, srcFile, destFile)
	return err
}

func (vm SoftLayerVM) downloadFile(virtualGuest datatypes.SoftLayer_Virtual_Guest, srcFile string, destFile string) error {
	err := vm.sshClient.DownloadFile(ROOT_USER_NAME, vm.getRootPassword(virtualGuest), virtualGuest.PrimaryIpAddress, srcFile, destFile)
	return err
}
