package com.example.controller;

import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;

@Controller
public class SomeController {
  @GetMapping("/helloGet")
  public String helloGet() {
    return "get!";
  }
}
