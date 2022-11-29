%global __os_install_post %{nil}
%define _version %(echo ${TAG%%%.*})
%define _release %(echo ${TAG##?.})
Name: resync
Version: %{_version}
Release: %{_release}
Summary: Resync is rsync and cron improved
License: Apache-2.0 license
Group: Applications/System
BuildArch: x86_64
BuildRoot: %{_tmppath}/%{name}-buildroot
%description
Resync is rsync and cron improved

%prep

%install
mkdir -p $RPM_BUILD_ROOT/usr/bin
mv ../SOURCES/apace_login $RPM_BUILD_ROOT/usr/bin/resync

mkdir -p $RPM_BUILD_ROOT/etc/systemd/system/multi-user.target.wants
mkdir -p $RPM_BUILD_ROOT/usr/lib/systemd/system
mv ../SOURCES/apace_login.service $RPM_BUILD_ROOT/usr/lib/systemd/system/resync.service

mkdir -p $RPM_BUILD_ROOT/etc/resync

%clean
rm -rf $RPM_BUILD_ROOT

%post
ln -s /usr/lib/systemd/system/resync.service /etc/systemd/system/multi-user.target.wants/resync.service
touch /etc/resync/resync.yaml
/usr/bin/systemctl enable resync.service
echo resync installed successfully

%preun
/usr/bin/systemctl disable resync.service

%files
%defattr(-,root,root)
/usr/bin/resync
/usr/lib/systemd/system/resync.service
/etc/systemd/system/multi-user.target.wants/resync.service
/etc/resync/resync.yaml
