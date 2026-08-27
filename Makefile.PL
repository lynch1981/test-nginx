use strict;
use lib '.';
use inc::Module::Install;

name            ('Test-Nginx');
license         ('bsd');
perl_version    ('5.010001');
repository      ('https://github.com/openresty/test-nginx');
all_from        ('lib/Test/Nginx.pm');

requires        ('Test::Base');
requires        ('Text::Diff');
requires        ('Test::LongString');
requires        ('IO::Socket');
requires        ('Time::HiRes');
requires        ('LWP::UserAgent');
requires        ('HTTP::Response');
requires        ('Fcntl');
requires        ('POSIX');
requires        ('IPC::Run');
requires        ('URI::Escape');
requires        ('IO::Select');
requires        ('File::Path');
requires        ('File::Temp');
requires        ('List::MoreUtils');
requires        ('List::Util');
requires        ('Encode');
requires        ('File::Find');
requires        ('IO::Socket::INET');

# Do not use use_test_base() here. It bundles Test::Builder without Test2,
# which can mix modules from different Test-Simple releases at runtime.
auto_install();

#tests('t/*.t t/*/*.t');

{
    package MY;
    sub postamble {
        my $self = shift;
        my $frag = '';
        if ($self->can('SUPER::postamble')) {
            $frag = $self->SUPER::postamble(@_);
            $frag = '' unless defined $frag;
        }
        return $frag unless _have_go();
        $frag .= <<'MAKE_FRAG';

.PHONY: utls-client
utls-client:
	cd go/utls-client && go build -o test-nginx-utls .

all :: utls-client
MAKE_FRAG
        return $frag;
    }

    sub _have_go {
        my $go = `command -v go 2>/dev/null`;
        return defined $go && $go =~ /\S/;
    }
}

WriteAll();
