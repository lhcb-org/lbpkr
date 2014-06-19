lbpkr
===

[![Build Status](https://drone.io/github.com/lhcb-org/lbpkr/status.png)](https://drone.io/github.com/lhcb-org/lbpkr/latest)

`lbpkr` is a `Go`-based re-implementation of `RpmInstall`.

## Installation

```go
$ go get github.com/lhcb-org/lbpkr
```

or, if you prefer the binary:
```sh
$ curl -O -L http://cern.ch/lhcbproject/dist/rpm/lbpkr && chmod +x ./lbpkr
$ ./lbpkr help
```

## Usage

### list available packages

```sh
$ lbpkr list LHCB
LHCBEXTERNALS_v68r0_x86_64_slc6_gcc48_opt-1.0.0-1-0
LHCB_v37r1-1.0.0-1-0
LHCB_v37r1_x86_64_slc6_gcc48_opt-1.0.0-1-0
LHCB_v37r3-1.0.0-1-0
LHCB_v37r3_x86_64_slc6_gcc48_dbg-1.0.0-1-0
LHCB_v37r3_x86_64_slc6_gcc48_opt-1.0.0-1-0
lbpkr INFO    Total matching: 6
```

### install a package (and its dependencies)

```sh
$ lbpkr install -type=atlas LCGCMT_LCGCMT_67b_i686_slc6_gcc47_opt-1-1
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

### list installed packages

```sh
$ lbpkr installed
AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-4
Boost-f9e91_1.55.0_python2.7_x86_64_slc6_gcc48_opt-1.0.0-4
CASTOR-9ccc5_2.1.13_6_x86_64_slc6_gcc48_opt-1.0.0-4
[...]
vdt-d9030_0.3.6_x86_64_slc6_gcc48_opt-1.0.0-4
xqilla-cefdd_2.2.4p1_x86_64_slc6_gcc48_opt-1.0.0-4
xrootd-3a806_3.2.7_x86_64_slc6_gcc48_opt-1.0.0-4
```

### find which package provides a file

```sh
$ lbpkr provides gaudirun.py
GAUDI_v25r1-1.0.0-1 (/opt/cern-sw/lhcb/GAUDI/GAUDI_v25r1/Gaudi/scripts/.svn/prop-base/gaudirun.py.svn-base)
GAUDI_v25r1_x86_64_slc6_gcc48_opt-1.0.0-1 (/opt/cern-sw/lhcb/GAUDI/GAUDI_v25r1/InstallArea/x86_64-slc6-gcc48-opt/scripts/gaudirun.py)
```

### list the dependencies of a given package

```sh
$ lbpkr deps ROOT-6ef81_5.34.18_x86_64_slc6_gcc48_opt
CASTOR-9ccc5_2.1.13_6_x86_64_slc6_gcc48_opt-1.0.0-4-0
GSL-a0511_1.10_x86_64_slc6_gcc48_opt-1.0.0-4-0
Python-31787_2.7.6_x86_64_slc6_gcc48_opt-1.0.0-5-0
Qt-f642c_4.8.4_x86_64_slc6_gcc48_opt-1.0.0-4-0
dcap-cdd28_2.47.7_1_x86_64_slc6_gcc48_opt-1.0.0-4-0
fftw-0c601_3.1.2_x86_64_slc6_gcc48_opt-1.0.0-4-0
gcc_4.8.1_x86_64_slc6-1.0.0-1-0
gfal-6fc75_1.13.0_0_x86_64_slc6_gcc48_opt-1.0.0-4-0
graphviz-a8340_2.28.0_x86_64_slc6_gcc48_opt-1.0.0-4-0
mysql-c4d2c_5.5.27_x86_64_slc6_gcc48_opt-1.0.0-4-0
oracle-e33b7_11.2.0.3.0_x86_64_slc6_gcc48_opt-1.0.0-4-0
sqlite-4b60e_3070900_x86_64_slc6_gcc48_opt-1.0.0-4-0
srm_ifce-be254_1.13.0_0_x86_64_slc6_gcc48_opt-1.0.0-4-0
xrootd-3a806_3.2.7_x86_64_slc6_gcc48_opt-1.0.0-4-0
```

### help

```sh
$ lbpkr help
lbpkr - installs software in MYSITEROOT directory.

Commands:

    check       check for RPM updates from the yum repository
    deps        list all deps RPM packages satisfying <name-pattern> [<version-pattern> [<release-pattern>]]
    install     install a RPM from the yum repository
    installed   list all installed RPM packages satisfying <name-pattern> [<version-pattern> [<release-pattern>]]
    list        list all RPM packages satisfying <name-pattern> [<version-pattern> [<release-pattern>]]
    provides    list all installed RPM packages providing the given file
    rpm         rpm passes through command-args to the RPM binary
    update      update RPMs from the yum repository
    version     print out script version

Use "lbpkr help <command>" for more information about a command.
```


## References

- https://twiki.cern.ch/twiki/bin/view/LHCb/InstallProjectWithRPM
- http://cern.ch/lhcbproject/GIT/RpmInstall.git
