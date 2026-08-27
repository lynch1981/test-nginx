# Optional live smoke: needs TEST_NGINX_TLS_CLIENT on PATH and nginx with ssl.
use lib 'lib';
use Test::Nginx::Socket::TLSClient;

BEGIN {
    unless (Test::Nginx::Util::have_tls_client()) {
        require Test::More;
        Test::More::plan(skip_all =>
            "TEST_NGINX_TLS_CLIENT not set or not executable");
    }

    my $bin = $Test::Nginx::Util::NginxBinary || 'nginx';
    my $ver = `$bin -V 2>&1`;
    if ($? != 0 || $ver !~ /http_ssl_module/) {
        require Test::More;
        Test::More::plan(skip_all => "nginx with http_ssl_module not found");
    }
}

repeat_each(1);
plan tests => repeat_each() * 2 * blocks();

no_shuffle();
run_tests();

__DATA__

=== TEST 1: GET via tls_client
--- config
    location = /t {
        return 200 'hello tls_client';
    }
--- request
    GET /t
--- response_body eval
"hello tls_client"
--- error_code: 200
