# RPM spec for the prebuilt PC Monitor binary. Built by scripts/package_linux.sh:
#   rpmbuild -bb --target <arch> --define "pkgversion X" --define "srcdir DIR" pc-monitor.spec
%global debug_package %{nil}
%global __strip /bin/true
%define _build_id_links none

Name:           pc-monitor
Version:        %{pkgversion}
Release:        1
Summary:        Monitor and control this computer from the PC Monitor dashboard
License:        GPL-3.0-only
URL:            https://github.com/garrettcorn/remote-monitor

%description
PC Monitor is a lightweight server that streams system telemetry (CPU, GPU,
memory, drives, network, displays) to the PC Monitor web dashboard and,
after PIN pairing, allows remote control (audio, buttons, power).

%install
mkdir -p %{buildroot}/usr/bin %{buildroot}/usr/share/applications %{buildroot}/usr/share/icons/hicolor/256x256/apps
install -m 0755 %{srcdir}/pc_monitor %{buildroot}/usr/bin/pc_monitor
install -m 0644 %{srcdir}/pc-monitor.desktop %{buildroot}/usr/share/applications/pc-monitor.desktop
install -m 0644 %{srcdir}/pc-monitor.png %{buildroot}/usr/share/icons/hicolor/256x256/apps/pc-monitor.png

%files
/usr/bin/pc_monitor
/usr/share/applications/pc-monitor.desktop
/usr/share/icons/hicolor/256x256/apps/pc-monitor.png
