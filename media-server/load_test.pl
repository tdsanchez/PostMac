#!/usr/bin/env perl
use strict;
use warnings;
use v5.40;

=head1 NAME

load_test.pl - Media Server Load Testing Script (Perl Implementation)

=head1 SYNOPSIS

    perl load_test.pl --url http://localhost:8080 --workers 5 --duration 60
    perl load_test.pl --scenario stress --workers 10 --duration 120
    perl load_test.pl --scenario api-only --requests 1000

=head1 DESCRIPTION

Perl implementation of load testing for media server. Validates:
- Template serialization bottleneck elimination
- Containerization readiness
- API endpoint performance
- Cross-language HTTP client behavior

=cut

use Getopt::Long qw(:config no_ignore_case bundling);
use Time::HiRes qw(time sleep);
use HTTP::Tiny;
use JSON::PP;
use URI::Encode qw(uri_encode);
use List::Util qw(min max sum shuffle);
use POSIX qw(strftime);

# Configuration
my %config = (
    url      => 'http://localhost:8080',
    workers  => 5,
    duration => 60,
    requests => undef,
    scenario => 'mixed',
    timeout  => 30,
    warmup   => 1,
);

# Results tracking
my %results = (
    total_requests      => 0,
    successful_requests => 0,
    failed_requests     => 0,
    broken_pipes        => 0,
    timeouts            => 0,
    total_duration_ms   => 0,
    min_duration_ms     => 1e9,
    max_duration_ms     => 0,
    errors_by_type      => {},
    requests_by_endpoint => {},
    durations           => [],
);

# Test data
my @categories;
my %files_by_category;

sub print_banner {
    print "\n" . "=" x 80 . "\n";
    print "MEDIA SERVER LOAD TEST (Perl $^V)\n";
    print "=" x 80 . "\n";
    printf "URL:       %s\n", $config{url};
    printf "Workers:   %d\n", $config{workers};
    printf "Duration:  %ds%s\n", $config{duration},
           (defined $config{requests} ? " (or until request count)" : "");
    printf "Scenario:  %s\n", $config{scenario};
    printf "Timeout:   %ds\n", $config{timeout};
    print "=" x 80 . "\n\n";
}

sub make_request {
    my ($http, $endpoint, $params) = @_;
    my $start = time();

    my $url = $config{url} . $endpoint;
    if ($params && %$params) {
        my @query = map { "$_=" . uri_encode($params->{$_}) } keys %$params;
        $url .= '?' . join('&', @query);
    }

    my $response = $http->get($url);
    my $duration_ms = (time() - $start) * 1000;

    return {
        endpoint    => $endpoint,
        duration_ms => $duration_ms,
        status_code => $response->{status},
        error       => $response->{success} ? undef : $response->{reason},
        content     => $response->{content},
    };
}

sub record_stat {
    my ($stat) = @_;

    $results{total_requests}++;
    $results{requests_by_endpoint}{$stat->{endpoint}}++;

    if ($stat->{error}) {
        $results{failed_requests}++;
        $results{errors_by_type}{$stat->{error}}++;

        if ($stat->{error} =~ /broken pipe/i) {
            $results{broken_pipes}++;
        }
        if ($stat->{error} =~ /timeout/i) {
            $results{timeouts}++;
        }
    } else {
        $results{successful_requests}++;
        $results{total_duration_ms} += $stat->{duration_ms};
        $results{min_duration_ms} = min($results{min_duration_ms}, $stat->{duration_ms});
        $results{max_duration_ms} = max($results{max_duration_ms}, $stat->{duration_ms});
        push @{$results{durations}}, $stat->{duration_ms};
    }
}

sub discover_categories {
    my ($http) = @_;

    # Use common test categories
    @categories = ('All', '5-★★★★★', '📁 Classico');
    print "✓ Using test categories: " . join(', ', @categories) . "\n";
    return @categories;
}

sub fetch_file_list {
    my ($http, $category) = @_;

    my $stat = make_request($http, '/api/filelist', { category => $category });

    if (!$stat->{error} && $stat->{status_code} == 200) {
        my $files = eval { decode_json($stat->{content}) };
        if ($files && ref($files) eq 'ARRAY') {
            $files_by_category{$category} = $files;
            printf "✓ Fetched %d files for category '%s'\n", scalar(@$files), $category;
            return $files;
        }
    }

    warn "⚠ Failed to fetch files for '$category'\n";
    return [];
}

sub viewer_request {
    my ($http, $category, $random_mode) = @_;

    my $files = $files_by_category{$category} || [];
    my $encoded_cat = uri_encode($category);

    if (!@$files) {
        return make_request($http, "/view/$encoded_cat");
    }

    my $file_path = $files->[int(rand(@$files))];
    my $params = { file => $file_path };
    $params->{random} = 'true' if $random_mode;

    return make_request($http, "/view/$encoded_cat", $params);
}

sub gallery_request {
    my ($http, $category, $page) = @_;
    my $encoded_cat = uri_encode($category);
    return make_request($http, "/tag/$encoded_cat", { page => $page || 1 });
}

sub api_filelist_request {
    my ($http, $category) = @_;
    return make_request($http, '/api/filelist', { category => $category });
}

sub homepage_request {
    my ($http) = @_;
    return make_request($http, '/');
}

sub mixed_workload {
    my ($http) = @_;

    return homepage_request($http) unless @categories;

    my $category = $categories[int(rand(@categories))];
    my $rand = rand(100);

    if ($rand < 40) {
        # 40% viewer requests
        return viewer_request($http, $category, int(rand(2)));
    } elsif ($rand < 70) {
        # 30% gallery requests
        return gallery_request($http, $category, int(rand(5)) + 1);
    } elsif ($rand < 90) {
        # 20% API requests
        return api_filelist_request($http, $category);
    } else {
        # 10% homepage
        return homepage_request($http);
    }
}

sub worker_thread {
    my ($scenario, $duration_seconds, $requests_count) = @_;

    my $http = HTTP::Tiny->new(
        timeout => $config{timeout},
        agent   => 'MediaServerLoadTest-Perl/1.0',
    );

    my @local_stats;
    my $start_time = time();
    my $request_count = 0;

    while (1) {
        # Check termination conditions
        my $elapsed = time() - $start_time;
        last if $duration_seconds > 0 && $elapsed >= $duration_seconds;
        last if defined($requests_count) && $request_count >= $requests_count;

        # Execute request based on scenario
        my $stat;
        if ($scenario eq 'mixed') {
            $stat = mixed_workload($http);
        } elsif ($scenario eq 'viewer-random') {
            my $cat = @categories ? $categories[int(rand(@categories))] : 'All';
            $stat = viewer_request($http, $cat, 1);
        } elsif ($scenario eq 'gallery') {
            my $cat = @categories ? $categories[int(rand(@categories))] : 'All';
            $stat = gallery_request($http, $cat);
        } elsif ($scenario eq 'api-only') {
            my $cat = @categories ? $categories[int(rand(@categories))] : 'All';
            $stat = api_filelist_request($http, $cat);
        } elsif ($scenario eq 'stress') {
            $stat = mixed_workload($http);
        } else {
            $stat = mixed_workload($http);
        }

        push @local_stats, $stat;
        $request_count++;

        # Add small delay for non-stress scenarios
        sleep(0.1) if $scenario ne 'stress' && $duration_seconds > 0;
    }

    return \@local_stats;
}

sub calculate_percentile {
    my ($sorted_array, $percentile) = @_;
    return 0 unless @$sorted_array;
    my $index = int($percentile * @$sorted_array);
    $index = @$sorted_array - 1 if $index >= @$sorted_array;
    return $sorted_array->[$index];
}

sub print_results {
    my ($elapsed_seconds) = @_;

    print "\n" . "=" x 80 . "\n";
    print "LOAD TEST RESULTS\n";
    print "=" x 80 . "\n";

    print "\n📊 Overall Statistics:\n";
    printf "  Total Requests:      %d\n", $results{total_requests};
    my $success_rate = $results{total_requests} > 0
        ? ($results{successful_requests} / $results{total_requests}) * 100
        : 0;
    printf "  Successful:          %d (%.2f%%)\n", $results{successful_requests}, $success_rate;
    printf "  Failed:              %d\n", $results{failed_requests};
    printf "  Broken Pipes:        %d %s\n", $results{broken_pipes},
           $results{broken_pipes} > 0 ? '⚠️  BOTTLENECK!' : '✅';
    printf "  Timeouts:            %d\n", $results{timeouts};
    printf "  Test Duration:       %.2fs\n", $elapsed_seconds;
    my $throughput = $elapsed_seconds > 0
        ? $results{total_requests} / $elapsed_seconds
        : 0;
    printf "  Throughput:          %.2f req/s\n", $throughput;

    if ($results{successful_requests} > 0) {
        print "\n⏱️  Response Times:\n";
        my $avg = $results{total_duration_ms} / $results{successful_requests};
        printf "  Average:             %.2f ms\n", $avg;
        printf "  Min:                 %.2f ms\n", $results{min_duration_ms};
        printf "  Max:                 %.2f ms\n", $results{max_duration_ms};

        # Calculate percentiles
        my @sorted_durations = sort { $a <=> $b } @{$results{durations}};
        if (@sorted_durations) {
            my $p50 = calculate_percentile(\@sorted_durations, 0.50);
            my $p95 = calculate_percentile(\@sorted_durations, 0.95);
            my $p99 = calculate_percentile(\@sorted_durations, 0.99);
            printf "  p50:                 %.2f ms\n", $p50;
            printf "  p95:                 %.2f ms\n", $p95;
            printf "  p99:                 %.2f ms\n", $p99;
        }
    }

    if (%{$results{requests_by_endpoint}}) {
        print "\n🎯 Requests by Endpoint:\n";
        for my $endpoint (sort { $results{requests_by_endpoint}{$b} <=>
                                 $results{requests_by_endpoint}{$a} }
                         keys %{$results{requests_by_endpoint}}) {
            printf "  %-30s %d\n", $endpoint, $results{requests_by_endpoint}{$endpoint};
        }
    }

    if (%{$results{errors_by_type}}) {
        print "\n❌ Errors by Type:\n";
        for my $error (sort { $results{errors_by_type}{$b} <=>
                             $results{errors_by_type}{$a} }
                      keys %{$results{errors_by_type}}) {
            printf "  %-30s %d\n", $error, $results{errors_by_type}{$error};
        }
    }

    print "\n" . "=" x 80 . "\n";

    # Verdict
    if ($results{broken_pipes} > 0) {
        print "⚠️  VERDICT: BROKEN PIPES DETECTED - Template serialization bottleneck still present!\n";
    } elsif ($success_rate >= 99.0 && ($results{successful_requests} == 0 ||
             $results{total_duration_ms} / $results{successful_requests} < 500)) {
        print "✅ VERDICT: EXCELLENT - System handles load gracefully!\n";
    } elsif ($success_rate >= 95.0) {
        print "✓ VERDICT: GOOD - System performs adequately under load\n";
    } else {
        print "⚠️  VERDICT: POOR - High error rate indicates issues\n";
    }
    print "=" x 80 . "\n\n";
}

# Parse command line options
GetOptions(
    'url=s'      => \$config{url},
    'workers=i'  => \$config{workers},
    'duration=i' => \$config{duration},
    'requests=i' => \$config{requests},
    'scenario=s' => \$config{scenario},
    'timeout=i'  => \$config{timeout},
    'no-warmup'  => sub { $config{warmup} = 0 },
    'help|h'     => \my $help,
) or die "Error parsing options!\n";

if ($help) {
    print <<'HELP';
Usage: perl load_test.pl [OPTIONS]

Options:
  --url URL          Base URL of media server (default: http://localhost:8080)
  --workers N        Number of concurrent workers (default: 5)
  --duration SEC     Test duration in seconds (default: 60)
  --requests N       Number of requests per worker (overrides duration)
  --scenario NAME    Test scenario: mixed, viewer-random, gallery, api-only, stress
  --timeout SEC      Request timeout in seconds (default: 30)
  --no-warmup        Skip warmup phase
  --help, -h         Show this help

Scenarios:
  mixed          Mixed workload (default) - homepage, gallery, viewer, API
  viewer-random  Focus on viewer with random mode (stress test serialization)
  gallery        Gallery page requests with pagination
  api-only       API endpoint testing (/api/filelist)
  stress         Aggressive stress test with no delays

Examples:
  perl load_test.pl --workers 5 --duration 60
  perl load_test.pl --scenario viewer-random --workers 10 --duration 30
  perl load_test.pl --scenario stress --workers 20 --duration 120
HELP
    exit 0;
}

# Main execution
print_banner();

my $http = HTTP::Tiny->new(
    timeout => $config{timeout},
    agent   => 'MediaServerLoadTest-Perl/1.0',
);

# Warmup phase
if ($config{warmup}) {
    print "🔥 Warmup phase...\n";
    discover_categories($http);
    for my $category (@categories[0..min(2, $#categories)]) {
        fetch_file_list($http, $category);
    }
    print "✓ Warmup complete\n\n";
}

# Start load test
printf "🚀 Starting load test with %d workers...\n", $config{workers};
my $start_time = time();

# Fork workers
my @pids;
my @temp_files;

for my $worker_id (1..$config{workers}) {
    my $temp_file = "/tmp/loadtest_perl_$$" . "_$worker_id.json";
    push @temp_files, $temp_file;

    my $pid = fork();
    die "Fork failed: $!" unless defined $pid;

    if ($pid == 0) {
        # Child process
        my $stats = worker_thread($config{scenario}, $config{duration}, $config{requests});

        # Write results to temp file
        open my $fh, '>', $temp_file or die "Cannot write $temp_file: $!";
        print $fh encode_json($stats);
        close $fh;

        exit 0;
    }

    push @pids, $pid;
}

# Wait for all workers
waitpid($_, 0) for @pids;

# Collect results
for my $temp_file (@temp_files) {
    next unless -f $temp_file;

    open my $fh, '<', $temp_file or next;
    my $content = do { local $/; <$fh> };
    close $fh;

    my $stats = eval { decode_json($content) };
    if ($stats && ref($stats) eq 'ARRAY') {
        record_stat($_) for @$stats;
    }

    unlink $temp_file;
}

my $elapsed_seconds = time() - $start_time;

# Print results
print_results($elapsed_seconds);

# Exit with appropriate code
exit($results{broken_pipes} > 0 ? 1 : 0);
