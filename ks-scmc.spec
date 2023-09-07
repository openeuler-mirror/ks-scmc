#
# spec file for KylinSec security container magic cube
#

%global debug_package %{nil}

Summary:        KylinSec security container magic cube
License:        LGPLv2+
Group:          Tools/Container/Docker

Name:           ks-scmc
Version:        1.0.0
Release:        1%{?dist}
URL:            http://gitlab.kylinos.com.cn/os/%{name}
Source0:        %{name}-%{version}.tar.gz

Summary:       KylinSec security container magic cube server package
BuildRequires: pkgconfig(systemd) git golang make coreutils protobuf-compiler systemd
Requires: docker-ce mysql5-server mysql5 coreutils bash socat systemd
        # cadvisor
        # influxdb
        # lsync csync2

%description
KylinSec security container magic cube provides simply, effecient and secure container management.

%prep
%autosetup -c -n %{name}-%{version}

%build
cd backend && make && cd -

%install
cd backend && make DESTDIR=$RPM_BUILD_ROOT install && cd -

%post
%systemd_post %{name}-agent.service
%systemd_post %{name}-controller.service
%systemd_post %{name}-authz.service

%preun
%systemd_preun %{name}-agent.service
%systemd_preun %{name}-controller.service
%systemd_preun %{name}-authz.service

%files
%defattr(-,root,root)
%{_bindir}/%{name}-server
%{_bindir}/%{name}-authz
%{_unitdir}/%{name}-agent.service
%{_unitdir}/%{name}-controller.service
%{_unitdir}/%{name}-authz.service
%dir %attr(755, root, root) %{_var}/lib/%{name}
%dir %attr(755, root, root) %{_var}/log/%{name}
%dir %attr(755, root, root) %{_sysconfdir}/%{name}/
%{_sysconfdir}/%{name}/access-container-gui
%config(noreplace) %{_sysconfdir}/%{name}/setup_config.sh
%config(noreplace) %{_sysconfdir}/%{name}/graphic_rc
%{_sysconfdir}/%{name}/database.sql

%changelog
* Wed Apr 20 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.0-1.ky3
- KYOS-F: initial setup for ks-scmc. (#48150)
