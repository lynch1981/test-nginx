# Unit tests for Test::Nginx::Socket::gen_utls_cmd_from_req
use lib 'lib';
use Test::Nginx::Socket tests => 8;

local $Test::Nginx::Util::UtlsBinary = '/opt/test-nginx-utls';
local $Test::Nginx::Util::ServerAddr = '127.0.0.1';
local $Test::Nginx::Util::ServerPortForClient = 1984;
local $Test::Nginx::Util::ServerName = 'localhost';
local $Test::Nginx::Util::UtlsClient = 'chrome';
local $Test::Nginx::Util::UtlsVerify = undef;
local $Test::Nginx::Util::Verbose = undef;
local $Test::Nginx::Util::UseHttp2 = undef;

sub block_named {
    my $name = shift;
    for my $b (blocks()) {
        return $b if $b->name eq $name;
    }
    die "no test block named $name";
}

sub cmd_for {
    my $name = shift;
    my $b = block_named($name);
    my $reqs = Test::Nginx::Socket::get_req_from_block($b);
    return Test::Nginx::Socket::gen_utls_cmd_from_req($b, $reqs->[0]);
}

sub flag_value {
    my ($cmd, $flag) = @_;
    for (my $i = 0; $i < $#$cmd; $i++) {
        return $cmd->[$i + 1] if $cmd->[$i] eq $flag;
    }
    return undef;
}

is_deeply(cmd_for('defaults'),
          ['/opt/test-nginx-utls',
           '--client', 'chrome',
           '--addr', '127.0.0.1:1984',
           '--sni', 'localhost',
           '--timeout', timeout(),
           '--http2', 'auto',
           '--insecure'],
          'default argv');

is_deeply(cmd_for('firefox sni'),
          ['/opt/test-nginx-utls',
           '--client', 'firefox',
           '--addr', '127.0.0.1:1984',
           '--sni', 'example.com',
           '--timeout', timeout(),
           '--http2', 'auto',
           '--insecure'],
          'utls_client and utls_sni');

{
    my $cmd = cmd_for('no http2');
    is(flag_value($cmd, '--http2'), 'never', '--- no_http2 sends --http2 never');
}

{
    my $cmd = cmd_for('require http2');
    is(flag_value($cmd, '--http2'), 'require', '--- http2 sends --http2 require');
}

{
    my $cmd = cmd_for('verify');
    ok(!grep { $_ eq '--insecure' } @$cmd, '--- utls_verify omits --insecure');
}

{
    my $cmd = cmd_for('alpn');
    is(flag_value($cmd, '--alpn'), 'h2,http/1.1', '--- utls_alpn becomes a comma list');
}

{
    my $cmd = cmd_for('timeout');
    is(flag_value($cmd, '--timeout'), 7, '--- timeout is passed in seconds');
}

{
    my $cmd = cmd_for('ipv6');
    is(flag_value($cmd, '--addr'), '[::1]:1984', 'IPv6 addresses are bracketed');
}

__DATA__

=== defaults
--- request
GET /t

=== firefox sni
--- utls_client: firefox
--- utls_sni: example.com
--- request
GET /t

=== no http2
--- no_http2
--- request
GET /t

=== require http2
--- http2
--- request
GET /t

=== verify
--- utls_verify
--- request
GET /t

=== alpn
--- utls_alpn: h2 http/1.1
--- request
GET /t

=== timeout
--- timeout: 7
--- request
GET /t

=== ipv6
--- server_addr_for_client: ::1
--- request
GET /t
