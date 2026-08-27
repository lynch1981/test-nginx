# Optional live smoke: needs the Go helper and an nginx with http_ssl_module.
use lib 'lib';
use Test::Nginx::Socket::UTLS;

BEGIN {
    my $helper = Test::Nginx::Util::resolve_utls_binary();
    if (!-x $helper) {
        require Test::More;
        Test::More::plan(skip_all => "uTLS helper not built ($helper)");
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

=== TEST 1: chrome GET over TLS
--- utls_client: chrome
--- config
    location = /t {
        return 200 'hello utls';
    }
--- request
    GET /t
--- response_body eval
"hello utls"
--- error_code: 200

=== TEST 2: golang GET over TLS
--- utls_client: golang
--- no_http2
--- config
    location = /t {
        return 200 'hello golang';
    }
--- request
    GET /t
--- response_body eval
"hello golang"
--- error_code: 200
