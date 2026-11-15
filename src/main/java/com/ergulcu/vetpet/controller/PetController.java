// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.controller;

import com.ergulcu.vetpet.dao.OwnerRepository;
import com.ergulcu.vetpet.dao.PetRepository;
import com.ergulcu.vetpet.model.Owner;
import com.ergulcu.vetpet.model.Pet;
import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.validation.BindingResult;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.ModelAttribute;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.server.ResponseStatusException;
import org.springframework.web.servlet.mvc.support.RedirectAttributes;

import java.util.List;

@Controller
@RequestMapping("/pet")
public class PetController {

    private final PetRepository petRepository;
    private final OwnerRepository ownerRepository;

    public PetController(PetRepository petRepository, OwnerRepository ownerRepository) {
        this.petRepository = petRepository;
        this.ownerRepository = ownerRepository;
    }

    /** The owner drop-down of the create and update forms. */
    @ModelAttribute("owners")
    public List<Owner> owners() {
        return ownerRepository.findAllByOrderByName();
    }

    @GetMapping("/list")
    public String list(@RequestParam(required = false) String name, Model model) {
        model.addAttribute("name", name);
        model.addAttribute("pets", name == null || name.isBlank()
                ? petRepository.findAllByOrderByName()
                : petRepository.findByNameContainingIgnoreCaseOrderByName(name.strip()));
        return "pet/list";
    }

    @GetMapping("/create")
    public String createForm(@ModelAttribute Pet pet) {
        return "pet/create";
    }

    @PostMapping("/create")
    public String create(@Valid @ModelAttribute Pet pet, BindingResult result, RedirectAttributes redirect) {
        if (result.hasErrors()) {
            return "pet/create";
        }
        pet.setId(null);
        petRepository.save(pet);
        redirect.addFlashAttribute("message", "Pet " + pet.getName() + " was created");
        return "redirect:/pet/list";
    }

    @GetMapping("/update/{id}")
    public String updateForm(@PathVariable long id, Model model) {
        model.addAttribute("pet", find(id));
        return "pet/update";
    }

    @PostMapping("/update/{id}")
    public String update(@PathVariable long id, @Valid @ModelAttribute Pet pet, BindingResult result,
                         RedirectAttributes redirect) {
        find(id);
        pet.setId(id);
        if (result.hasErrors()) {
            return "pet/update";
        }
        petRepository.save(pet);
        redirect.addFlashAttribute("message", "Pet " + pet.getName() + " was updated");
        return "redirect:/pet/list";
    }

    @PostMapping("/delete/{id}")
    public String delete(@PathVariable long id, RedirectAttributes redirect) {
        Pet pet = find(id);
        petRepository.delete(pet);
        redirect.addFlashAttribute("message", "Pet " + pet.getName() + " was deleted");
        return "redirect:/pet/list";
    }

    private Pet find(long id) {
        return petRepository.findById(id)
                .orElseThrow(() -> new ResponseStatusException(HttpStatus.NOT_FOUND, "Pet " + id + " not found"));
    }
}
