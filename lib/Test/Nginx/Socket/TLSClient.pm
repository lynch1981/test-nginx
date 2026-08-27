package Test::Nginx::Socket::TLSClient;

use Test::Nginx::Socket -Base;

our $VERSION = '0.33';

# Using this module opts the file into the tls_client backend.
# The actual binary still comes from TEST_NGINX_TLS_CLIENT or --- tls_client.
$Test::Nginx::Util::TlsClientMode = 1;

1;
__END__

=encoding utf-8

=head1 NAME

Test::Nginx::Socket::TLSClient - Socket-backed tests that send requests via an external TLS client

=head1 SYNOPSIS

    use Test::Nginx::Socket::TLSClient;

    repeat_each(1);
    plan tests => repeat_each() * 2 * blocks();

    run_tests();

    __DATA__

    === TEST 1: custom TLS client
    --- tls_client: my-tls-client
    --- tls_client_options: --client chrome
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
over TLS by running an external client command (not shipped here; same idea
as C<curl> for HTTP/2). Plain HTTP still uses the in-process socket backend.

It is equivalent to setting C<TEST_NGINX_TLS_CLIENT> for the file.
C<--- no_tls_client> still falls back to the plain TCP / curl backends.

The command name must still be given via C<TEST_NGINX_TLS_CLIENT> or
C<--- tls_client>. Nginx is configured with C<listen ssl> automatically.

See L<Test::Nginx::Socket/tls_client> for sections and environment variables.

=head1 AUTHOR

Yichun "agentzh" Zhang (章亦春) C<< <agentzh@gmail.com> >>, OpenResty Inc.

=head1 COPYRIGHT & LICENSE

Copyright (c) 2009-2025, Yichun Zhang C<< <agentzh@gmail.com> >>, OpenResty Inc.

This module is licensed under the terms of the BSD license.

=cut
