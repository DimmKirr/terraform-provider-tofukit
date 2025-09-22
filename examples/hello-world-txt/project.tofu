resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  # Single file scaffold - no dependencies needed
  scaffold {
    path = "hello.txt"
    content = <<-EOF
hello world
EOF
  }

  scaffold {
    path = "hello2.txt"
    content = <<-EOF
hello world2
EOF
  }


  scaffold {
    path = "demo/hello4.txt"
    content = <<-EOF
hello world4
EOF
  }

#   scaffold {
#     path = "hello3.txt"
#     content = <<-EOF
# hello world3
# EOF
#   }

}
