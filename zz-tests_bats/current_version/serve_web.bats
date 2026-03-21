#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"
}

teardown() {
	if [[ -n "${web_server_PID:-}" ]]; then
		kill "$web_server_PID" 2>/dev/null || true
	fi

	chflags_nouchg
}

# bats file_tags=user_story:serve_web

function serve_web_types { # @test
	start_web_server .

	run http --body GET "http://localhost:$web_port/types"
	assert_success
	assert_output --partial '"object-id"'
}

function serve_web_tags { # @test
	start_web_server .

	run http --body GET "http://localhost:$web_port/tags"
	assert_success
}

function serve_web_objects { # @test
	start_web_server .

	run http --body GET "http://localhost:$web_port/objects"
	assert_success
	assert_output --partial '['
}

function serve_web_types_index { # @test
	start_web_server .

	run http --body GET "http://localhost:$web_port/types_index"
	assert_success
	assert_output --partial '"total_words"'
}

function serve_web_single_type { # @test
	start_web_server .

	run http --body GET "http://localhost:$web_port/types/md"
	assert_success
	assert_output --partial '"object-id"'
	assert_output --partial '"objects-resource"'
}

function serve_web_type_objects { # @test
	start_web_server .

	run http --body GET "http://localhost:$web_port/types/md/objects"
	assert_success
	assert_output --partial '!md'
}

function serve_web_query { # @test
	start_web_server .

	run http --body GET "http://localhost:$web_port/query/:z"
	assert_success
	assert_output --partial '"object-id"'
}

function serve_web_not_found { # @test
	start_web_server .

	run http --body GET "http://localhost:$web_port/objects/nonexistent/zettel"
	assert_output --partial '"error"'
}

function serve_web_cors_header { # @test
	start_web_server .

	run http --headers GET "http://localhost:$web_port/types"
	assert_success
	assert_output --partial 'Access-Control-Allow-Origin'
}
