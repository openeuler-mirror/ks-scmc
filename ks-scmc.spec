#
# spec file for KylinSec security container magic cube
#

%global debug_package %{nil}

Summary:        KylinSec security container magic cube
License:        LGPLv2+
Group:          Tools/Container/Docker

Name:           ks-scmc
Version:        1.0
Release:        1%{?dist}
URL:            http://gitlab.kylinos.com.cn/va/%{name}
Source0:        %{name}-%{version}.tar.gz

%package server
Summary:       KylinSec security container magic cube server package
BuildRequires: pkgconfig(systemd) git golang make coreutils protobuf-compiler systemd
Requires: docker-engine mysql5-server mysql5 coreutils bash socat systemd
        # cadvisor
        # influxdb
        # lsync csync2
%description server
KylinSec security container magic cube provides simply, effecient and secure container management.

%package client
Summary:       KylinSec security container magic cube client package
BuildRequires: cmake coreutils protobuf-compiler protobuf-devel gcc-c++
BuildRequires: grpc-devel make kiran-log-qt5-devel qt5-qtbase-devel
BuildRequires: qt5-linguist zlog-devel
Requires: grpc kiran-log-qt5 protobuf qt5-qtbase zlog openssh-clients mate-terminal

%description client
KylinSec security container magic cube provides simply, effecient and secure container management.


%description
KylinSec security container magic cube provides simply, effecient and secure container management.

%prep
%autosetup -c -n %{name}-%{version}

%build
cd backend && make env && make && cd -
cd client && make && cd -

%install
cd backend && make DESTDIR=$RPM_BUILD_ROOT install && cd -
cd client/build/ && make DESTDIR=$RPM_BUILD_ROOT install && cd -

%post server
%systemd_post docker.service
%systemd_post mysqld.service
systemctl enable --now docker.service
systemctl enable --now mysqld.service

bash /etc/%{name}/setup_config.sh /etc/%{name}/server.toml
%systemd_post %{name}-agent.service
%systemd_post %{name}-controller.service
systemctl enable --now %{name}-agent.service
systemctl enable --now %{name}-controller.service

%preun server
%systemd_preun %{name}-agent.service
%systemd_preun %{name}-controller.service

%files client
%{_bindir}/%{name}-client
%dir %attr(755, root, root) %{_datadir}/%{name}/
%dir %attr(755, root, root) %{_datadir}/%{name}/translations/
%dir %attr(755, root, root) %{_datadir}/%{name}/icons/
%dir %attr(755, root, root) %{_datadir}/applications/
%{_datadir}/%{name}/translations/%{name}.zh_CN.qm
%{_datadir}/%{name}/icons/%{name}-logo.png
%{_datadir}/applications/%{name}.desktop

%files server
%defattr(-,root,root)
%{_bindir}/%{name}-server
%{_unitdir}/%{name}-agent.service
%{_unitdir}/%{name}-controller.service
%dir %attr(755, root, root) %{_var}/lib/%{name}
%dir %attr(755, root, root) %{_var}/log/%{name}
%dir %attr(755, root, root) %{_sysconfdir}/%{name}/
%{_sysconfdir}/%{name}/access-container-gui
%config(noreplace) %{_sysconfdir}/%{name}/setup_config.sh
%config(noreplace) %{_sysconfdir}/%{name}/graphic_rc
%{_sysconfdir}/%{name}/database.sql

%changelog
* Tue Feb 15 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.0-1.ky3
- KYOS-F: Node management backend RPC and Qt user interface.
- KYOS-F: Container management backend RPC and Qt user interface.
