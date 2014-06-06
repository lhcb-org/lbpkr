lbpkr
===

[![Build Status](https://drone.io/github.com/lhcb-org/lbpkr/status.png)](https://drone.io/github.com/lhcb-org/lbpkr/latest)

`lbpkr` is a `Go`-based re-implementation of `RpmInstall`.

## Installation

```go
$ go get github.com/lhcb-org/lbpkr
```

## Usage

### list available packages

```sh
$ lbpkr list LHCB
lbpkr INFO    RPM DB in "/opt/cern-sw/var/lib/rpm"
repo INFO    checking availability of backend [RepositorySQLiteBackend]
repo INFO    repository [repo] - chosen backend [*yum.RepositorySQLiteBackend]
repo INFO    checking availability of backend [RepositorySQLiteBackend]
repo INFO    updating the RPM database for RepositorySQLiteBackend
repo INFO    repository [lcg] - chosen backend [*yum.RepositorySQLiteBackend]
repo INFO    checking availability of backend [RepositorySQLiteBackend]
repo INFO    updating the RPM database for RepositorySQLiteBackend
repo INFO    repository [lhcbold] - chosen backend [*yum.RepositorySQLiteBackend]
repo INFO    checking availability of backend [RepositorySQLiteBackend]
repo INFO    updating the RPM database for RepositorySQLiteBackend
repo INFO    repository [lhcb] - chosen backend [*yum.RepositorySQLiteBackend]
lbpkr INFO    rpm: Found "/bin/rpm"
LHCB_v34r2-1.0.0-1
LHCB_v34r2_x86_64_slc5_gcc43_opt-1.0.0-1
LHCB_v34r2_x86_64_slc5_gcc46_opt-1.0.0-1
LHCB_v35r1p1_x86_64_slc5_gcc43_opt-1.0.0-1
LHCB_v35r1p1-1.0.0-1
LHCB_v35r0-1.0.0-1
LHCB_v35r0_x86_64_slc5_gcc43_opt-1.0.0-1
lbpkr INFO    Total matching: 7
```

### install a package (and its dependencies)

```sh
$ lbpkr install -type=atlas LCGCMT_LCGCMT_67b_i686_slc6_gcc47_opt-1-1
lbpkr INFO    RPM DB in "/opt/cern-sw/var/lib/rpm"
lbpkr INFO    Initializing RPM db
repo INFO    checking availability of backend [RepositorySQLiteBackend]
repo INFO    updating the RPM database for RepositorySQLiteBackend
repo INFO    repository [repo] - chosen backend [*yum.RepositorySQLiteBackend]
lbpkr INFO    rpm: Found "/bin/rpm"
lbpkr INFO    installing RPM LCGCMT_LCGCMT_67b_i686_slc6_gcc47_opt 1 1
lbpkr INFO    installing LCGCMT_LCGCMT_67b_i686_slc6_gcc47_opt and dependencies
lbpkr INFO    found 31 RPMs to install:
lbpkr INFO    	[001/031] AtlasSetup-00.03.74-1
lbpkr INFO    	[002/031] CLHEP_1_9_4_7_i686_slc6_gcc47_opt-1-1
lbpkr INFO    	[003/031] CASTOR_2_1_13_6_i686_slc6_gcc47_opt-1-1
lbpkr INFO    	[004/031] ROOT_5_34_13_i686_slc6_gcc47_opt-1-1
lbpkr INFO    	[005/031] Expat_2_0_1_i686_slc6_gcc47_opt-1-1
lbpkr INFO    	[006/031] GCCXML_0_9_0_20120309p2_i686_slc6_gcc47_opt-1-1
lbpkr INFO    	[007/031] Boost_1_53_0_python2_7_i686_slc6_gcc47_opt-1-1
[...]
lbpkr INFO    downloading http://atlas-computing.web.cern.ch/atlas-computing/links/reposDirectory/lcg/slc6/yum//noarch/AIDA_3_2_1_noarch-1-1.noarch.rpm to /opt/cern-sw/tmp/AIDA_3_2_1_noarch-1-1.rpm
```

### help

```sh
$ lbpkr help
lbpkr - installs software in MYSITEROOT directory.

Commands:

    install     install a RPM from the yum repository
    list        list all RPM packages satisfying <name-pattern> [<version-pattern> [<release-pattern>]]
    version     print out script version

Use "lbpkr help <command>" for more information about a command.
```


## References

- https://twiki.cern.ch/twiki/bin/view/LHCb/InstallProjectWithRPM
- http://cern.ch/lhcbproject/GIT/RpmInstall.git
