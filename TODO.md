# TODO list

- [x] make driver-scan lighter adding a service level to manage MountDisplay/monutinfo and make it more testable. The service should just return the list of all devices hiding all complexity but making filtering or configuration possible. We shoudl use interfaces.

- add unit/integration tests to driver-scan

- improve readme.md to make it more oss compatible moving all lsblk detail in another documentation file

- [x] add Makefile

- implement a first prototype of driver-mounter with uuid as input parameter, following all convention used for driver-scan and using a well know golang library to skip complexity. This cli should mount a disk identified by uuid in a default mount point creating a subfolder. the mount point can be passed as paramter.

- implement a first prototype of driver-init following all convention used for driver-scan and using a well know golang library to skip complexity. This cli should partion/format with ext4 or a configurable FS an available disk after a scan (use a service used by driver-scan too). Check the best option to partion or format (library or exec program).
