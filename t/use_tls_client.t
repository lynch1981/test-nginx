# Unit tests for Test::Nginx::Util::use_tls_client
use lib 'lib';
use Test::Nginx::Socket tests => 8;

sub block_named {
    my $name = shift;
    for my $b (blocks()) {
        return $b if $b->name eq $name;
    }
    die "no test block named $name";
}

{
    my $b = block_named('explicit tls_client');
    ok(Test::Nginx::Util::use_tls_client($b), '--- tls_client enables the backend');
    ok(Test::Nginx::Util::use_tls_client($b), 'second call uses the cached result');
}

{
    local $Test::Nginx::Util::TlsClientMode = 1;
    local $Test::Nginx::Util::TlsClient = 'my-client';
    my $b = block_named('no_tls_client wins');
    ok(!Test::Nginx::Util::use_tls_client($b), '--- no_tls_client disables a global run');
}

{
    local $Test::Nginx::Util::TlsClientMode = 1;
    local $Test::Nginx::Util::TlsClient = 'my-client';
    my $b = block_named('env request');
    ok(Test::Nginx::Util::use_tls_client($b), 'TEST_NGINX_TLS_CLIENT enables blocks with --- request');
}

{
    local $Test::Nginx::Util::TlsClientMode = 1;
    local $Test::Nginx::Util::TlsClient = 'my-client';
    local $SIG{__WARN__} = sub {};
    my $b = block_named('env raw_request');
    ok(!Test::Nginx::Util::use_tls_client($b), 'raw_request is skipped in global tls_client mode');
}

{
    local $Test::Nginx::Util::TlsClientMode = 1;
    local $Test::Nginx::Util::TlsClient = 'my-client';
    local $SIG{__WARN__} = sub {};
    my $b = block_named('env pipelined');
    ok(!Test::Nginx::Util::use_tls_client($b), 'pipelined_requests is skipped in global tls_client mode');
}

{
    my $b = block_named('plain request');
    local $Test::Nginx::Util::TlsClientMode = 0;
    ok(!Test::Nginx::Util::use_tls_client($b), 'tls_client is off by default');
}

{
    local $Test::Nginx::Util::TlsClientMode = 1;
    local $Test::Nginx::Util::TlsClient = 'my-client';
    my $b = block_named('env no request');
    ok(!Test::Nginx::Util::use_tls_client($b), 'global tls_client skips blocks without --- request');
}

__DATA__

=== explicit tls_client
--- tls_client: my-client
--- request
GET /t

=== no_tls_client wins
--- no_tls_client
--- request
GET /t

=== env request
--- request
GET /t

=== env raw_request
--- raw_request eval
"GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"

=== env pipelined
--- pipelined_requests eval
["GET /a", "GET /b"]

=== plain request
--- request
GET /t

=== env no request
--- config
    location = /t { return 200; }
