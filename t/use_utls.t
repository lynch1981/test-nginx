# Unit tests for Test::Nginx::Util::use_utls
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
    my $b = block_named('explicit utls');
    ok(Test::Nginx::Util::use_utls($b), '--- utls enables the backend');
    ok(Test::Nginx::Util::use_utls($b), 'second call uses the cached result');
}

{
    local $Test::Nginx::Util::UseUtls = 1;
    my $b = block_named('no_utls wins');
    ok(!Test::Nginx::Util::use_utls($b), '--- no_utls disables a global uTLS run');
}

{
    local $Test::Nginx::Util::UseUtls = 1;
    my $b = block_named('env request');
    ok(Test::Nginx::Util::use_utls($b), 'TEST_NGINX_USE_UTLS enables blocks with --- request');
}

{
    local $Test::Nginx::Util::UseUtls = 1;
    local $SIG{__WARN__} = sub {};
    my $b = block_named('env raw_request');
    ok(!Test::Nginx::Util::use_utls($b), 'raw_request is skipped in global uTLS mode');
}

{
    local $Test::Nginx::Util::UseUtls = 1;
    local $SIG{__WARN__} = sub {};
    my $b = block_named('env pipelined');
    ok(!Test::Nginx::Util::use_utls($b), 'pipelined_requests is skipped in global uTLS mode');
}

{
    my $b = block_named('plain request');
    local $Test::Nginx::Util::UseUtls = 0;
    ok(!Test::Nginx::Util::use_utls($b), 'uTLS is off by default');
}

{
    local $Test::Nginx::Util::UseUtls = 1;
    my $b = block_named('env no request');
    ok(!Test::Nginx::Util::use_utls($b), 'global uTLS skips blocks without --- request');
}

__DATA__

=== explicit utls
--- utls
--- request
GET /t

=== no_utls wins
--- no_utls
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
