#
# spec file for KylinSec security container magic cube
#

%global debug_package %{nil}

%global pkg_name ks-scmc
%global pkg_ver 1.1.1
%global pkg_rel 5

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
Requires: docker-ce cadvisor influxdb opensnitch
Requires: ks-license-client ks-license-gui-client
%if "%{kylin_version}" == "3.3"
Requires: kernel-kek-devel kernel-kek
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

getent group %{name} >/dev/null || groupadd -r %{name}
mkdir -p /home/%{name}
getent passwd %{name} >/dev/null || useradd -r -d /home/%{name} -g %{name} -c "container terminal connect user" %{name}
chown %{name}:%{name} /home/%{name}
echo "kylin.2023" | passwd --stdin %{name}

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
%{_bindir}/%{name}-encrypt
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
%{_sysconfdir}/sudoers.d/ks-scmc-sudoers

%changelog
* Fri Nov 17 2023 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.1.1-2.ky3
- KYOS-F: allow update container envs/mounts. (#21151)
- KYOS-F: fix loong64 arch building issue.

* Wed Nov 8 2023 Gongping Zheng <zhenggongping@kylinos.com.cn> - 1.1.1-1.ky3
- KYOS-F: Rebuild for KY3.3-6-PG-ks-scmc-1.1.

* Sat Aug 20 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.4-2.ky3
- KYOS-F: Fix container backup occasionally failure. (#57408)

* Thu Jul 7 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.4-1.ky3
- KYOS-F: Store backup job in filesystem (#57408)
- KYOS-F: rpc ListRuntime support fuzzing matching (#57412)
- KYOS-F: rpc ContainerList fix nil pointer reference (#58474)
- KYOS-F: Check image file before downloading (#59395)
- KYOS-F: User default timeout set to 30min.

* Thu Jul 7 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.3-1.ky3
- KYOS-F: Move content of backend/ to top dir (#57217)
- KYOS-F: Fix createNode error handling, optimize cpu count getter (#56642 #57410)
- KYOS-F: Fix prevent docker-registry connection leak (#57436)
- KYOS-F: Fix ImageUpload and ImageRemove error (#57359 #56808)
- KYOS-F: Fix container list cpu/memory usage (#57413)

* Mon Jun 27 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.2-3.ky3
- KYOS-F: Fix string args checking (#56642 #56792)
- KYOS-F: Fix warn log format, allow only one cron task (#55827)
- KYOS-F: Fix error handling when removing container backup (#56714)
- KYOS-F: Optimize image management (#56592 #56808)
- KYOS-F: Optimize deploy scripts (#56690)
- KYOS-F: Optmize method of calculate CPU usage (#56698)

* Tue Jun 14 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.2-2.ky3
- KYOS-F: Fix manipulate opensnitch rules errors (#56692)
- KYOS-F: update password no permission error (#56693)

* Tue Jun 7 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.2-1.ky3
- KYOS-F: Fix cannot login with user secadm (#55825)
- KYOS-F: Fix container list display wrong start time, disk usage (#55882 #56020)
- KYOS-F: Fix container cannot set memory limit (#55824)
- KYOS-F: Fix remove container failed returns no error (#56029)
- KYOS-F: Container basic/security setting need different permission (#55988)
- KYOS-F: Fix resume container backup error because of ip conflict (#55887)

* Sat May 21 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.1-1.ky3
- KYOS-F: security settings (file protection, process whitelist). (#54582 #54770)
- KYOS-F: security settings (net-process whitelist, iptables rules). (#54288 #53553)
- KYOS-F: security settings (container control authz plugin). (#54542)
- KYOS-F: runtime/warn log rpcs. (#52978)
- KYOS-F: user/role/permission management rpcs. (#53473)

* Wed Apr 20 2022 Haotian Chen <chenhaotian@kylinos.com.cn> - 1.0.0-1.ky3
- KYOS-F: initial setup for ks-scmc. (#48150)
