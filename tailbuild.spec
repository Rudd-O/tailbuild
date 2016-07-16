%define debug_package %{nil}

Name:           tailbuild
Version:        0.1.2
Release:        1%{?dist}
Summary:        A Jenkins build log tailer

License:        GPLv3+
URL:            https://github.com/Rudd-O/tailbuild
Source0:	Source0: https://github.com/Rudd-O/%{name}/archive/{%version}.tar.gz#/%{name}-%{version}.tar.gz

BuildRequires:  go

%description
tailbuild is a small command line utility to tail Jenkins build logs in real-time.

%prep
%setup -q

%build
# variables must be kept in sync with install
make DESTDIR=$RPM_BUILD_ROOT BINDIR=%{_bindir}

%install
rm -rf $RPM_BUILD_ROOT
# variables must be kept in sync with build
make install DESTDIR=$RPM_BUILD_ROOT BINDIR=%{_bindir}

%files
%attr(0755, root, root) %{_bindir}/tailbuild
%doc README.md

%changelog
* Mon Mar 28 2016 Manuel Amador (Rudd-O) <rudd-o@rudd-o.com
- Initial release
