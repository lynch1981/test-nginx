package Test::Nginx::Socket::UTLS;

use Test::Nginx::Socket -Base;

our $VERSION = '0.33';

# Using this module opts the file into the uTLS backend, matching
# TEST_NGINX_USE_UTLS=1. Individual blocks can still set --- no_utls.
unless (defined $ENV{TEST_NGINX_USE_UTLS} && $ENV{TEST_NGINX_USE_UTLS} eq '0') {
    $Test::Nginx::Util::UseUtls = 1;
}

1;
__END__

=encoding utf-8

=head1 NAME

Test::Nginx::Socket::UTLS - Socket-backed test scaffold that speaks TLS as a browser

=head1 SYNOPSIS

    use Test::Nginx::Socket::UTLS;

    repeat_each(1);
    plan tests => repeat_each() * 2 * blocks();

    run_tests();

    __DATA__

    === TEST 1: chrome
    --- utls_client: chrome
    --- config
        location = /t {
            return 200 'ok';
        }
    --- request
        GET /t
    --- response_body eval
    "ok"
    --- error_code: 200

=head1 DESCRIPTION

This module subclasses L<Test::Nginx::Socket> and sends every test request
over TLS using the C<test-nginx-utls> helper (uTLS ClientHello parroting).

It is equivalent to setting C<TEST_NGINX_USE_UTLS=1> for the file.
C<--- no_utls> still falls back to the plain TCP / curl backends.

See L<Test::Nginx::Socket/utls> for sections and environment variables.

=head1 AUTHOR

Yichun "agentzh" Zhang (章亦春) C<< <agentzh@gmail.com> >>, OpenResty Inc.

=head1 COPYRIGHT & LICENSE

Copyright (c) 2009-2025, Yichun Zhang C<< <agentzh@gmail.com> >>, OpenResty Inc.

This module is licensed under the terms of the BSD license.

=cut
