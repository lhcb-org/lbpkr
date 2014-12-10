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
LHCBEXTERNALS_v68r0_x86_64_slc6_gcc48_opt-1.0.0-1
LHCB_v37r1-1.0.0-1
LHCB_v37r1_x86_64_slc6_gcc48_opt-1.0.0-1
LHCB_v37r3-1.0.0-1
LHCB_v37r3_x86_64_slc6_gcc48_dbg-1.0.0-1
LHCB_v37r3_x86_64_slc6_gcc48_opt-1.0.0-1
lbpkr INFO    Total matching: 6
```

### install a (list of) package(s) (and its dependencies)

```sh
$ lbpkr install \
   LCG_70_AIDA_3.2.1_x86_64_slc6_gcc48_opt \
   LCG_70root6_AIDA_3.2.1_x86_64_slc6_gcc48_opt

lbpkr INFO    installing RPMs [LCG_70_AIDA_3.2.1_x86_64_slc6_gcc48_opt LCG_70root6_AIDA_3.2.1_x86_64_slc6_gcc48_opt]
lbpkr INFO    installing LCG_70_AIDA_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-12 and dependencies
lbpkr INFO    installing LCG_70root6_AIDA_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-12 and dependencies
lbpkr INFO    found 8 RPMs to install:
lbpkr INFO    	[001/008] AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-7
lbpkr INFO    	[002/008] LCGMeta_70_externals_x86_64_slc6_gcc48_opt-1.0.0-12
lbpkr INFO    	[003/008] LCGMeta_70root6_externals_x86_64_slc6_gcc48_opt-1.0.0-13
lbpkr INFO    	[004/008] LCG_70_AIDA_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-12
lbpkr INFO    	[005/008] LCG_70_gcc_4.8.1_x86_64_slc6-1.0.0-71
lbpkr INFO    	[006/008] LCG_70root6_AIDA_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-12
lbpkr INFO    	[007/008] LCG_70root6_gcc_4.8.1_x86_64_slc6-1.0.0-16
lbpkr INFO    	[008/008] gcc_4.8.1_x86_64_slc6-1.0.0-1
lbpkr INFO    [001/008] downloaded http://cern.ch/service-spi/external/rpms/lcg/LCG_70root6_gcc_4.8.1_x86_64_slc6-1.0.0-16.noarch.rpm
lbpkr INFO    [002/008] downloaded http://cern.ch/service-spi/external/rpms/lcg/LCG_70_gcc_4.8.1_x86_64_slc6-1.0.0-71.noarch.rpm
lbpkr INFO    [003/008] downloaded http://cern.ch/service-spi/external/rpms/lcg/LCGMeta_70root6_externals_x86_64_slc6_gcc48_opt-1.0.0-13.noarch.rpm
lbpkr INFO    [004/008] downloaded http://cern.ch/service-spi/external/rpms/lcg/LCG_70_AIDA_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-12.noarch.rpm
lbpkr INFO    [005/008] downloaded http://cern.ch/service-spi/external/rpms/lcg/LCGMeta_70_externals_x86_64_slc6_gcc48_opt-1.0.0-12.noarch.rpm
lbpkr INFO    [006/008] downloaded http://cern.ch/service-spi/external/rpms/lcg/LCG_70root6_AIDA_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-12.noarch.rpm
lbpkr INFO    [007/008] downloaded http://cern.ch/service-spi/external/rpms/lcg/AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-7.noarch.rpm
lbpkr INFO    [008/008] downloaded http://cern.ch/service-spi/external/rpms/lcg/gcc_4.8.1_x86_64_slc6-1.0.0-1.noarch.rpm
lbpkr INFO    installing [8] RPMs...
Preparing...                ##################################################
AIDA-3fe9f_3.2.1_x86_64_slc6##################################################
gcc_4.8.1_x86_64_slc6       ##################################################
LCG_70root6_gcc_4.8.1_x86_64##################################################
LCG_70_gcc_4.8.1_x86_64_slc6##################################################
LCGMeta_70_externals_x86_64_##################################################
LCGMeta_70root6_externals_x8##################################################
LCG_70root6_AIDA_3.2.1_x86_6##################################################
LCG_70_AIDA_3.2.1_x86_64_slc##################################################
```

### install a project

```sh
$ lbpkr install-project -platforms=all GAUDI v25r5
lbpkr INFO    installing project GAUDI v25r5
lbpkr INFO    installing project name="GAUDI" version="v25r5" for archs=[x86_64_slc6_gcc48_dbg x86_64_slc6_gcc48_opt]
lbpkr INFO    installing GAUDI_v25r5_x86_64_slc6_gcc48_dbg-1.0.0-1 and dependencies
lbpkr INFO    installing GAUDI_v25r5_x86_64_slc6_gcc48_opt-1.0.0-1 and dependencies
lbpkr INFO    found 186 RPMs to install:
lbpkr INFO    	[001/186] AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_dbg-1.0.0-12
lbpkr INFO    	[002/186] AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_opt-1.0.0-7
lbpkr INFO    	[003/186] Boost-f9e91_1.55.0_python2.7_x86_64_slc6_gcc48_dbg-1.0.0-12
[...]
lbpkr INFO    [001/184] downloaded http://cern.ch/service-spi/external/rpms/lcg/LCG_70root6_srm_ifce_1.13.0_0_x86_64_slc6_gcc48_opt-1.0.0-12.noarch.rpm
[...]
lbpkr INFO    [186/186] downloaded http://cern.ch/service-spi/external/rpms/lcg/ROOT-90bb1_6.02.01_x86_64_slc6_gcc48_dbg-1.0.0-12.noarch.rpm
lbpkr INFO    installing [186] RPMs...
Preparing...                ##################################################
[...]
GAUDI_v25r5_x86_64_slc6_gcc4##################################################
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
CASTOR-9ccc5_2.1.13_6_x86_64_slc6_gcc48_opt-1.0.0-4
GSL-a0511_1.10_x86_64_slc6_gcc48_opt-1.0.0-4
Python-31787_2.7.6_x86_64_slc6_gcc48_opt-1.0.0-5
Qt-f642c_4.8.4_x86_64_slc6_gcc48_opt-1.0.0-4
dcap-cdd28_2.47.7_1_x86_64_slc6_gcc48_opt-1.0.0-4
fftw-0c601_3.1.2_x86_64_slc6_gcc48_opt-1.0.0-4
gcc_4.8.1_x86_64_slc6-1.0.0-1
gfal-6fc75_1.13.0_0_x86_64_slc6_gcc48_opt-1.0.0-4
graphviz-a8340_2.28.0_x86_64_slc6_gcc48_opt-1.0.0-4
mysql-c4d2c_5.5.27_x86_64_slc6_gcc48_opt-1.0.0-4
oracle-e33b7_11.2.0.3.0_x86_64_slc6_gcc48_opt-1.0.0-4
sqlite-4b60e_3070900_x86_64_slc6_gcc48_opt-1.0.0-4
srm_ifce-be254_1.13.0_0_x86_64_slc6_gcc48_opt-1.0.0-4
xrootd-3a806_3.2.7_x86_64_slc6_gcc48_opt-1.0.0-4
```

### dump the depencency graph of installed RPMs

```sh
$ lbpkr dep-grap -o graph.dot GAUDI
GAUDI_v25r2-1.0.0-1
GAUDI_v25r2_x86_64_slc6_gcc48_opt-1.0.0-1
$ cat graph.dot
digraph rpms {
        "GAUDI_v25r2_x86_64_slc6_gcc48_opt-1.0.0-1"->"LHCBEXTERNALS_v68r0_x86_64_slc6_gcc48_opt-1.0.0-1";
        "GAUDI_v25r2_x86_64_slc6_gcc48_opt-1.0.0-1"->"GAUDI_v25r2-1.0.0-1";
        "GAUDI_v25r2-1.0.0-1" [ epoch="0", name="GAUDI_v25r2", release="1", version="1.0.0" ];
        "GAUDI_v25r2_x86_64_slc6_gcc48_opt-1.0.0-1" [ epoch="0", name="GAUDI_v25r2_x86_64_slc6_gcc48_opt", release="1", version="1.0.0" ];
        "LHCBEXTERNALS_v68r0_x86_64_slc6_gcc48_opt-1.0.0-1" [ epoch="0", name="LHCBEXTERNALS_v68r0_x86_64_slc6_gcc48_opt", release="1", version="1.0.0" ];

}

## dump the full graph rooted in GAUDI_v25r2_x86_64_slc6_gcc48_opt
$ lbpkr dep-graph -o graph.dot -maxdepth=-1 GAUDI_v25r2_x86_64_slc6_gcc48_opt
```

### manage yum repositories

```sh

# list current yum repositories
$ lbpkr repo-ls
lcg: "http://cern.ch/service-spi/external/rpms/lcg" (enabled)
lhcb: "http://cern.ch/lhcbproject/dist/rpm/lhcb" (enabled)
lhcbext: "http://cern.ch/lhcbproject/dist/rpm/lcg" (enabled)

# add a test repo
$ lbpkr repo-add my-repo /tmp/somewhere

# list
$ lbpkr repo-ls
lcg: "http://cern.ch/service-spi/external/rpms/lcg" (enabled)
lhcb: "http://cern.ch/lhcbproject/dist/rpm/lhcb" (enabled)
lhcbext: "http://cern.ch/lhcbproject/dist/rpm/lcg" (enabled)
my-repo: "/tmp/somewhere" (enabled)

# remove the test repo
$ lbpkr repo-rm my-repo

# list
$ lbpkr repo-ls
lcg: "http://cern.ch/service-spi/external/rpms/lcg" (enabled)
lhcb: "http://cern.ch/lhcbproject/dist/rpm/lhcb" (enabled)
lhcbext: "http://cern.ch/lhcbproject/dist/rpm/lcg" (enabled)
```

### help

```sh
$ lbpkr help
lbpkr - installs software in MYSITEROOT directory.

Commands:

    check           check for RPM updates from the yum repository
    dep-graph       dump the DOT graph of installed RPM packages
    deps            list deps of RPM packages
    install         install a (list of) RPM(s) from the yum repository
    install-project install-project a whole project from the yum repository
    installed       list installed RPM packages
    list            list RPM packages
    provides        list all installed RPM packages providing the given file
    repo-add        add a repository
    repo-ls         list repositories
    repo-rm         remove a repository
    rm              remove a RPM from the yum repository
    rpm             pass through command-args to the RPM binary
    self            admin/internal operations for lbpkr
    update          update RPMs from the yum repository
    version         print out script version

Use "lbpkr help <command>" for more information about a command.
```


## References

- https://twiki.cern.ch/twiki/bin/view/LHCb/InstallProjectWithRPM
- http://cern.ch/lhcbproject/GIT/RpmInstall.git
