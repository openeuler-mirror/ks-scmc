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
%else
Requires: mysql5-server
%endif

%description
KylinSec security container magic cube provides simply, effecient and secure container management.

%prep
%setup -q -n %{name}-%{version}

%build
make && cd -

%install
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn
make DESTDIR=$RPM_BUILD_ROOT install && cd -

%post
%systemd_post %{name}-agent.service
%systemd_post %{name}-controller.service
%systemd_post %{name}-authz.service

# 仅在组不存在时创建组
if ! getent group %{name} >/dev/null; then
    groupadd -r %{name}
fi

