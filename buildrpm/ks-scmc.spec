#
# spec file for KylinSec security container magic cube
#

%global debug_package %{nil}

%global pkg_name ks-scmc
%global pkg_ver 1.3.0
%global pkg_rel 1

Summary:        KylinSec security container magic cube
License:        No License
Group:          Tools/Container/Docker

Name:           %{pkg_name}
Version:        %{pkg_ver}
Release:        %{pkg_rel}%{?dist}
URL:            http://gitlab.kylinos.com.cn/os/%{name}
Source0:        %{name}-%{version}.tar.gz

Summary:       KylinSec security container magic cube server package
BuildRequires: pkgconfig(systemd) git golang make coreutils protobuf-compiler systemd

Requires: coreutils bash socat systemd lsyncd rsync xinetd keepalived
Requires: docker-ce cadvisor influxdb
Requires: ks-license-client ks-license-gui-client
%if "%{kylin_version}" == "3.3"
Requires: mysql-community-server
