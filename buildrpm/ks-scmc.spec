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

# 仅在用户不存在时执行创建用户及相关操作
if ! getent passwd %{name} >/dev/null; then
    mkdir -p /home/%{name}
    useradd -r -d /home/%{name} -g %{name} -c "container terminal connect user" %{name}
    chown %{name}:%{name} /home/%{name}
    echo "kylin.2023" | passwd --stdin %{name}
fi

# getent group %{name} >/dev/null || groupadd -r %{name}
# mkdir -p /home/%{name}
# getent passwd %{name} >/dev/null || useradd -r -d /home/%{name} -g %{name} -c "container terminal connect user" %{name}
# chown %{name}:%{name} /home/%{name}
# echo "kylin.2023" | passwd --stdin %{name}

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
%{_bindir}/%{name}-sign
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
%{_sysconfdir}/%{name}/create_stdadm_r.json
%{_sysconfdir}/%{name}/init_users.sh
%{_sysconfdir}/sudoers.d/ks-scmc-sudoers

%changelog
* Tue Mar 24 2026 weidong<weidong@kylinsec.com.cn> - 1.3.0-1.ky3
- Fix container tampering alert. (b:117553)
- Fix container alert. (b:117555)

* Wed Mar 18 2026 yubin<yubin@kylinsec.com.cn> - 1.2.1-21.ky3
- Fix the issue of incorrect log reporting. (b:121661)
- resolve cpu usage for node alert. (b:128337)
- Fix missing monitoring data for containers after node reboot. (b:126766)
- Disk usage size and alerts. (b:126733)

* Fri Mar 6 2026 yubin<yubin@kylinsec.com.cn> - 1.2.1-20.ky3
- Fix the issue of incorrect log reporting. (b:121661)
- Fix issue where image name and tag are missing during image export. (b:120897)
- Invalid images cannot be loaded. (b:121487)
- Fix the issue where disk usage shows 0. (b.120970)

* Mon Feb 9 2026 yubin<yubin@kylinsec.com.cn> - 1.2.1-19.ky3
- Add X server access for launching GUI apps from desktops. (b:117434)
- add disk size limit. (b:116496)
- Support the running of GUI programs. (b:117434)
- Signature feature development. (b:116462)
- Do not calculate the SHA256 of images (b:116428)
- add validation for alert resource usage threshold values (b:116431)

* Thu Jan 22 2026 yubin<yubin@kylinsec.com.cn> - 1.2.1-18.ky3
