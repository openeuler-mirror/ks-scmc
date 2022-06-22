#
# spec file for KylinSec security container magic cube
#

%global debug_package %{nil}

Summary:        KylinSec security container magic cube
License:        No License
Group:          Tools/Container/Docker

Name:           ks-scmc
Version:        1.0.2
Release:        3
URL:            http://gitlab.kylinos.com.cn/os/%{name}
Source0:        %{name}-%{version}.tar.gz

BuildRequires: pkgconfig(systemd) git golang make coreutils protobuf-compiler systemd
Requires: docker-ce coreutils bash socat systemd lsyncd rsync xinetd keepalived

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

# write version info
mkdir -p %{_datadir}/ks-scmc
echo %{version}-%{release} > %{_datadir}/ks-scmc/ks-scmc.version
chmod 0744 %{_datadir}/ks-scmc/ks-scmc.version > /dev/null || :

%preun
%systemd_preun %{name}-agent.service
%systemd_preun %{name}-controller.service
%systemd_preun %{name}-authz.service

rm -rf %{_datadir}/ks-scmc

%files
%defattr(-,root,root)
%{_bindir}/%{name}-server
%{_bindir}/%{name}-authz
%{_bindir}/%{name}-user
%{_unitdir}/%{name}-agent.service
%{_unitdir}/%{name}-controller.service
%{_unitdir}/%{name}-authz.service
%dir %attr(755, root, root) %{_var}/lib/%{name}
%dir %attr(755, root, root) %{_var}/log/%{name}
%dir %attr(755, root, root) %{_sysconfdir}/%{name}/
%config(noreplace) %{_sysconfdir}/%{name}/server.toml
%{_sysconfdir}/%{name}/access-container-gui
%{_sysconfdir}/%{name}/graphic_rc
%{_sysconfdir}/%{name}/setup_db.sh
%{_sysconfdir}/%{name}/setup_agent.sh
%{_sysconfdir}/%{name}/database.sql
%{_sysconfdir}/%{name}/create_network.sh
%{_sysconfdir}/%{name}/sync_image.sh
%{_sysconfdir}/%{name}/keepalived.sh
%{_sysconfdir}/%{name}/mysql_double_master.sh
%{_sysconfdir}/%{name}/create_audadm.json
%{_sysconfdir}/%{name}/create_secadm.json
%{_sysconfdir}/%{name}/create_sysadm.json
%{_sysconfdir}/%{name}/create_audadm_r.json
%{_sysconfdir}/%{name}/create_secadm_r.json
%{_sysconfdir}/%{name}/create_sysadm_r.json
%{_sysconfdir}/%{name}/init_users.sh

%changelog
* Sat May 21 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.1-1.ky3
- KYOS-F: security settings (file protection, process whitelist). (#54582 #54770)
- KYOS-F: security settings (net-process whitelist, iptables rules). (#54288 #53553)
- KYOS-F: security settings (container control authz plugin). (#54542)
- KYOS-F: runtime/warn log rpcs. (#52978)
- KYOS-F: user/role/permission management rpcs. (#53473)

* Wed Apr 20 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.0-1.ky3
- KYOS-F: initial setup for ks-scmc. (#48150)
